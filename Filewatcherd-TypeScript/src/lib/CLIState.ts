/*******************************************************************************
* Copyright (c) 2019, 2020 IBM Corporation and others.
* All rights reserved. This program and the accompanying materials
* are made available under the terms of the Eclipse Public License v2.0
* which accompanies this distribution, and is available at
* http://www.eclipse.org/legal/epl-v20.html
*
* Contributors:
*     IBM Corporation - initial API and implementation
*******************************************************************************/

import * as child_process from "child_process";
import * as path from "path";
import * as log from "./Logger";
import { convertAbsoluteUnixStyleNormalizedPathToLocalFile } from "./PathUtils";
import { ProjectToWatch } from "./ProjectToWatch";

/* The purpose of this class is to call the cwctl project sync command, in order to allow the
 * Codewind CLI to detect and communicate file changes to the server.
 *
 * This class will ensure that only one instance of the cwctl project sync command is running
 * at a time, per project.
 *
 * For automated testing, if the `MOCK_CWCTL_INSTALLER_PATH` environment variable is specified, a mock cwctl command
 * written in Java (as a runnable JAR) can be used to test this class.
 */
export class CLIState {

    private _isProcessActive: boolean = false;
    private _isRequestWaiting: boolean = false;

    private readonly _projectId: string;

    /* Absolute time, in unix epoch msecs, at which the last cwctl command was initiated. */
    private _timestamp: number = 0;

    private readonly _installerPath: string;

    private readonly _projectPath: string;

    /** For automated testing only */
    private readonly _mockInstallerPath: string | undefined;

    /** For automated testing only */
    private _lastDebugPtwSeen: ProjectToWatch | undefined;

    constructor(projectId: string, installerPath: string, projectPath: string) {
        this._projectId = projectId;
        this._installerPath = installerPath;
        this._projectPath = projectPath;

        this._mockInstallerPath = process.env.MOCK_CWCTL_INSTALLER_PATH;
    }

    public onFileChangeEvent(projectCreationTimeInAbsoluteMsecsParam: number | undefined, debugPtw: ProjectToWatch | undefined) {

        if (!this._projectPath || this._projectPath.trim().length === 0) {
            log.error("Project path passed to CLIState is empty, so ignoring file change event.");
            return;
        }

        if (debugPtw) {
            this._lastDebugPtwSeen = debugPtw;
        }

        // This, along with callCLIAsync(), ensures that only one instance of `project sync` is running at a time.
        if (this._isProcessActive) {
            this._isRequestWaiting = true;
        } else {
            this._isProcessActive = true;
            this._isRequestWaiting = false;

            // Ensure that timestamp is updated with PCT, but only if timestamp is 0,
            // AND there isn't a process running, AND pct is non-null.
            {
                const debugOldTimestampValue = this._timestamp;

                // We only update the timestamp when 'callCLI' is true, because we don't want to
                // step on the toes of another running CLI process (and that one will probably
                // update the timestamp on it's own, with a more recent value, then ours)

                // Update the timestamp to the project creation value, but ONLY IF it is zero.
                if (projectCreationTimeInAbsoluteMsecsParam && projectCreationTimeInAbsoluteMsecsParam !== 0
                    && this._timestamp === 0) {

                    this._timestamp = projectCreationTimeInAbsoluteMsecsParam;

                    log.info("Timestamp updated from " + debugOldTimestampValue + " to " + this._timestamp
                        + " from project creation time.");

                }
            }

            this.callCLIAsync(this._lastDebugPtwSeen); // Do not await here
        }
    }

    private async callCLIAsync(debugPtw: ProjectToWatch | undefined) {

        const DEBUG_FAKE_CMD_OUTPUT = false; // Enable this for debugging purposes.

        // This try block works in tandem with onFileChangeEvent() to ensure that only one instance of the
        // 'project sync' command is running at a time.
        try {

            let result;

            if (DEBUG_FAKE_CMD_OUTPUT) {
                log.info("Faking a call to CLI with params " + this._timestamp + " " + this._projectId);
                result = {
                    errorCode: 0,
                    output: new Date().getTime().toString(),
                    spawnTime: new Date(),
                };

            } else {
                // Call CLI and wait for result
                result = await this.runProjectCommand(debugPtw);
            }

            if (result) {
                if (result.errorCode !== 0) {
                    // The stdout/stderr output will already have been printed, so we only output the error code here.
                    log.severe("Non-zero error code from installer: " + result.errorCode)
                } else {
                    // Success, so update the tiemstamp to the process start time.
                    this._timestamp = result.spawnTime.getTime();
                    log.info("Updating timestamp to latest: " + this._timestamp);
                }
            }

        } catch (e) {
            // Log, handle, then bury the exception
            log.severe("Unexpected exception from CLI", e);
        }

        this._isProcessActive = false;

        // If another file change list occurred during the last invocation, then start another one.
        if (this._isRequestWaiting) {
            this.onFileChangeEvent(undefined, undefined);
        }
    }

    private async runProjectCommand(debugPtw: ProjectToWatch | undefined): Promise<IRunProjectReturn> {

        const executableDir = path.dirname(this._installerPath);

        let firstArg = this._installerPath;

        let args: string[];

        const lastTimestamp = this._timestamp;

        if (!this._mockInstallerPath || this._mockInstallerPath.trim().length === 0) {
            // Normal call to `cwctl project sync`

            // Example:
            // cwctl project sync -p /Users/tobes/workspaces/git/eclipse/codewind/codewind-workspace/lib5 \
            //      -i b1a78500-eaa5-11e9-b0c1-97c28a7e77c7 -t 12345
            args = ["--insecure", "project", "sync", "-p", this._projectPath, "-i", this._projectId, "-t",
                "" + lastTimestamp];
        } else {

            if (!debugPtw) {
                throw new Error("debugPtw ProjectToWatch object was not defined, but is required when debug is enabled");
            }

            // The filewatcher is being run in an automated test scenario: we will now run a
            // mock version of cwctl that simulates the project sync command. This mock
            // version takes slightly different parameters.

            // Convert filesToWatch to absolute paths
            const convertedFilesToWatch = [];
            for (const fileToWatch of debugPtw.filesToWatch) {
                const val = convertAbsoluteUnixStyleNormalizedPathToLocalFile(fileToWatch);
                convertedFilesToWatch.push(val);
            }

            // Create a simplified version of the project to watch JSON.
            const simplifiedPtw = {
                filesToWatch: convertedFilesToWatch,
                ignoredFilenames: debugPtw.ignoredFilenames,
                ignoredPaths: debugPtw.ignoredPaths,
            };

            const base64Json = new Buffer(JSON.stringify(simplifiedPtw)).toString("base64");

            args = ["-jar", this._mockInstallerPath, "-p", this._projectPath, "-i", this._projectId, "-t",
                "" + lastTimestamp, "-projectJson", base64Json];
            firstArg = "java";
        }

        let debugStr = "";
        {
            for (const arg of args) {
                debugStr += "[" + arg + "] ";
            }

            log.info("Calling cwctl project sync with: { " + debugStr + "}");
        }

        return new Promise<any>((resolve, reject) => {

            const spawnTime = new Date();

            const child = child_process.spawn(firstArg, args, {
                cwd: executableDir,
            });

            child.on("error", (err) => {
                return reject(err);
            });

            let outStr = "";
            let errStr = "";
            child.stdout.on("data", (chunk) => { outStr += chunk.toString(); });
            child.stderr.on("data", (chunk) => { errStr += chunk.toString(); });

            child.on("close", (code: number | null) => {

                if (code == null) {
                    // this happens in SIGTERM case, not sure what else may cause it
                    log.debug(`Installer 'project sync' did not exit normally`);
                } else if (code !== 0) {
                    log.error("Error running 'project sync' installer command: " + debugStr);
                    outStr = outStr || "No output";
                    errStr = errStr || "No output"
                    log.error("- cwctl stdout: {{ " + outStr + " }}");
                    log.error("- cwctl stderr: {{ " + errStr + " }} ");

                    const result: IRunProjectReturn = {
                        errorCode: code,
                        output: outStr + errStr,
                        spawnTime,
                    };

                    resolve(result);
                } else {

                    const result: IRunProjectReturn = {
                        errorCode: code,
                        output: outStr,
                        spawnTime,
                    };

                    log.info("Successfully ran installer command: " + debugStr);
                    log.info("- cwctl output: {{ " + outStr + errStr + " }}");
                    resolve(result);
                }
            });
        });
    }

}

/** Return value from the runProjectCommand promise. */
interface IRunProjectReturn {
    spawnTime: Date;
    errorCode: number;
    output: string;
}
