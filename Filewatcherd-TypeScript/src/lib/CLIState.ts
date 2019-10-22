/*******************************************************************************
* Copyright (c) 2019 IBM Corporation and others.
* All rights reserved. This program and the accompanying materials
* are made available under the terms of the Eclipse Public License v2.0
* which accompanies this distribution, and is available at
* http://www.eclipse.org/legal/epl-v20.html
*
* Contributors:
*     IBM Corporation - initial API and implementation
*******************************************************************************/

import * as child_process from "child_process";
import * as fs from "fs";
import * as os from "os";
import * as path from "path";
import * as readline from "readline";
import * as log from "./Logger";

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

    /* Absolute time, in unix epoch msecs, at which the last cwctl command was initiated. Note this is
     * different from the actual value we provide to cwctl. */
    private _timestamp: number = 0;

    private readonly _installerPath: string;

    private readonly _projectPath: string;

    /** For automated testing only */
    private readonly _mockInstallerPath: string;

    constructor(projectId: string, installerPath: string, projectPath: string) {
        this._projectId = projectId;
        this._installerPath = installerPath;
        this._projectPath = projectPath;

        this._mockInstallerPath = process.env.MOCK_CWCTL_INSTALLER_PATH;
    }

    public onFileChangeEvent() {

        if (!this._projectPath || this._projectPath.trim().length === 0) {
            log.error("Project path passed to CLIState is empty, so ignoring file change event.");
            return;
        }

        // This, along with callCLIAsync(), ensures that only one instance of `project sync` is running at a time.
        if (this._isProcessActive) {
            this._isRequestWaiting = true;
        } else {
            this._isProcessActive = true;
            this._isRequestWaiting = false;
            this.callCLIAsync(); // Do not await here
        }
    }

    private async callCLIAsync() {

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
                result = await this.runProjectCommand();
            }

            if (result) {
                if (result.errorCode !== 0) {
                    log.severe("Non-zero error code from installer: " + (result && result.output ? result.output : ""));
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
            this.onFileChangeEvent();
        }
    }

    private async runProjectCommand(): Promise<IRunProjectReturn> {

        const executableDir = path.dirname(this._installerPath);

        let firstArg = this._installerPath;

        let args: string[];

        // Convert the absolute timestamp to # of msecs since start of last run.
        const adjustedTimestamp = (new Date()).getTime() - this._timestamp;

        if (!this._mockInstallerPath || this._mockInstallerPath.trim().length === 0) {
            // Example:
            // cwctl project sync -p /Users/tobes/workspaces/git/eclipse/codewind/codewind-workspace/lib5 \
            //      -i b1a78500-eaa5-11e9-b0c1-97c28a7e77c7 -t 12345
            args = ["project", "sync", "-p", this._projectPath, "-i", this._projectId, "-t", "" + adjustedTimestamp];
        } else {
            args = ["-jar", this._mockInstallerPath, "-p", this._projectPath, "-i", this._projectId, "-t",
                "" + adjustedTimestamp];
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
                    log.error("Error running 'project sync' installer command " + errStr);
                    outStr = outStr || "No output";
                    errStr = errStr || "Unknown error " + args.join(" ");
                    log.error("Stdout:" + outStr);
                    log.error("Stderr:" + errStr);

                    const result: IRunProjectReturn = {
                        errorCode: code,
                        output: outStr + errStr,
                        spawnTime,
                    };

                    reject(result);
                } else {

                    const result: IRunProjectReturn = {
                        errorCode: code,
                        output: outStr,
                        spawnTime,
                    };

                    log.info("Successfully ran installer command: " + debugStr);
                    log.info("Output:" + outStr); // TODO: Convert to DEBUG once everything matures.
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
