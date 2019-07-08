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

import { HttpGetStatusThread } from "./HttpGetStatusThread";
import * as PathUtils from "./PathUtils";
import { ProjectObject } from "./ProjectObject";
import { ProjectToWatch } from "./ProjectToWatch";

import { WatchEventEntry } from "./WatchEventEntry";

import { ChangedFileEntry } from "./ChangedFileEntry";
import { HttpPostOutputQueue } from "./HttpPostOutputQueue";
import { PathFilter } from "./PathFilter";
import { ProjectToWatchFromWebSocket } from "./ProjectToWatchFromWebSocket";
import { WebSocketManagerThread } from "./WebSocketManagerThread";

import * as request from "request-promise-native";
import { DebugTimer } from "./DebugTimer";
import { ExponentialBackoffUtil } from "./ExponentialBackoffUtil";
import { IWatchService } from "./IWatchService";
import * as log from "./Logger";

export class FileWatcher {

    private readonly _projectsMap: Map<string /* project id*/, ProjectObject>;

    private readonly _baseUrl: string;
    private readonly _wsBaseUrl: string;

    private readonly _getStatusThread: HttpGetStatusThread;

    private readonly _watchService: IWatchService;

    private readonly _outputQueue: HttpPostOutputQueue;

    private readonly _webSocketManager: WebSocketManagerThread;

    private readonly _clientUuid: string;

    private _disposed: boolean = false;

    constructor(urlParam: string, watchService: IWatchService, clientUuid: string) {

        this._clientUuid = clientUuid;

        this._watchService = watchService;
        this._watchService.setParent(this);

        this._projectsMap = new Map<string, ProjectObject>();

        this._baseUrl = PathUtils.stripTrailingSlash(urlParam);

        this._outputQueue = new HttpPostOutputQueue(this._baseUrl);

        let calculatedWsUrl = this._baseUrl;
        calculatedWsUrl = calculatedWsUrl.replace("http://", "ws://");
        calculatedWsUrl = calculatedWsUrl.replace("https://", "wss://");
        this._wsBaseUrl = calculatedWsUrl;

        this._getStatusThread = new HttpGetStatusThread(this._baseUrl, this);

        this._webSocketManager = new WebSocketManagerThread(this._wsBaseUrl, this);
        this._webSocketManager.queueEstablishConnection();

        const debugTimer = new DebugTimer(this);
    }

    public updateFileWatchStateFromGetRequest(projectsToWatch: ProjectToWatch[]) {
        if (this._disposed) { return; }

        // First we remove the old projects

        log.info("Examining file watch state from GET request");

        const removedProjects = new Array<ProjectToWatch>();
        const projectIdInHttpResult = new Map<string, boolean>();

        for (const entry of projectsToWatch) {

            if (projectIdInHttpResult.has(entry.projectId)) {
                log.severe("Multiple projects in the project list share the same project ID: " + entry.projectId);
            }

            projectIdInHttpResult.set(entry.projectId, true);
        }

        // For each of the projects in the local state map, if they aren't found
        // in the HTTP GET result, then they have been removed.
        for (const [_, value] of this._projectsMap) {
            if (!projectIdInHttpResult.has(value.projectToWatch.projectId)) {
                removedProjects.push(value.projectToWatch);
            }
        }

        removedProjects.forEach( (e) => {
            this.removeSingleProjectToWatch(e);
        });

        // Next we process the new or updated projects

        for (const entry of projectsToWatch) {
            this.createOrUpdateProjectToWatch(entry);
        }

    }

    public receiveNewWatchEventEntry(watchEntry: WatchEventEntry, receivedAtInEpochMsecs: number) {
        if (this._disposed) { return; }

        let projectsToWatch = new Array<ProjectToWatch>();

        for (const [key, value] of this._projectsMap) {
            if (value) {
                projectsToWatch.push(value.projectToWatch);
            }
        }

        /*
         * Sort projectsToWatch by length, descending (this handles the case where a
         * parent, and it's child, are both managed by us.
         */
        projectsToWatch = projectsToWatch.sort((a, b) => {
            return b.pathToMonitor.length - a.pathToMonitor.length;
        });

        // This will be the absolute path on the local drive
        const fullLocalPath: string = watchEntry.absolutePathWithUnixSeparators;

        let match: ProjectToWatch = null;

        for (const ptw of projectsToWatch) {

            // See if this watch event is related to the project
            if (fullLocalPath.startsWith(ptw.pathToMonitor)) {
                match = ptw;
                break;
            }
        }

        if (!match) {
            log.severe("Could not find matching project");
            return;
        }

        // Any event type (create, modify, delete) is acceptable.

        // Filter it, then pass it to FilechangeEventBatchUtil

        const filter = new PathFilter(match);

        // Path will necessarily already have lowercase Windows drive letter, if
        // applicable.

        let path = PathUtils.convertAbsolutePathWithUnixSeparatorsToProjectRelativePath(
            watchEntry.absolutePathWithUnixSeparators, match.pathToMonitor);

        // let path = watchEntry.absolutePathWithUnixSeparators;

        // if (!path.startsWith(match.pathToMonitor)) {
        //     // This shouldn't happen, and is thus severe
        //     const msg = "Watch event '" + path + "' does not match project path '" + match.pathToMonitor + "'";
        //     log.severe(msg);
        //     return;
        // }

        // // Strip project parent directory from path:
        // // If pathToMonitor is: /home/user/codewind/project
        // // and watchEventPath is: /home/user/codewind/project/some-file.txt
        // // then this will convert watchEventPath to /some-file.txt
        // path = path.substring(match.pathToMonitor.length);

        // // If the event is occurring on the root path
        if (!path || path.length === 0) {
            path = "/";
        }

        if (match.ignoredPaths && filter.isFilteredOutByPath(path)) {
            log.debug("Filtering out " + path + " by path.");
            return;
        }
        if (match.ignoredFilenames && filter.isFilteredOutByFilename(path)) {
            log.debug("Filtering out " + path + " by filename.");
            return;
        }

        const newEntry = new ChangedFileEntry(path, watchEntry.eventType, receivedAtInEpochMsecs, watchEntry.directory);

        const po = this._projectsMap.get(match.projectId);
        if (po) {
            const e = [newEntry];
            po.batchUtil.addChangedFiles(e);
        } else {
            log.severe("Could not locate event processing for project id " + match.projectId);
        }
    }

    public sendBulkFileChanges(projectId: string, mostRecentEntryTimestamp: number, base64Compressed: string[]) {
        if (this._disposed) { return; }

        this._outputQueue.addToQueue(projectId, mostRecentEntryTimestamp, base64Compressed);
    }

    public updateFileWatchStateFromWebSocket(ptwList: ProjectToWatchFromWebSocket[]) {
        if (this._disposed) { return; }

        log.info("Examining file watch state update from WebSocket");

        for (const ptw of ptwList) {
            if (ptw.changeType === "add" || ptw.changeType === "update") {
                this.createOrUpdateProjectToWatch(ptw);
            } else if (ptw.changeType === "delete") {
                this.removeSingleProjectToWatch(ptw);
            } else {
                log.severe("Unepected changeType in message: " + JSON.stringify(ptw));
            }
        }

    }

    public refreshWatchStatus() {
        if (this._disposed) { return; }

        this._getStatusThread.queueStatusUpdate();
    }

    public async sendWatchResponseAsync(successParam: boolean, ptw: ProjectToWatch): Promise<void> {
        if (this._disposed) { return; }

        const backoffUtil = ExponentialBackoffUtil.getDefaultBackoffUtil(4000);

        let sendSuccess = false;

        while (!sendSuccess) {

            const payload: any = {
                success: successParam,
            };

            const options = {
                body: payload,
                json: true,
                resolveWithFullResponse: true,
                timeout: 20000,
            };

            const url = this._baseUrl + "/api/v1/projects/" + ptw.projectId + "/file-changes/"
                + ptw.projectWatchStateId + "/status?clientUuid=" + this._clientUuid;

            log.info("Issuing PUT request to '" + url + "' for " + ptw.projectId);
            try {
                const result = await request.put(url, options);

                if (result.statusCode !== 200) {
                    log.error("Unexpected error code " + result.statusCode
                        + " from '" + url + "' for " + ptw.projectId);

                    sendSuccess = false;
                } else {
                    log.debug("PUT request to '" + url + "' succeeded for " + ptw.projectId);
                    sendSuccess = true;
                }

            } catch (err) {
                log.error("Unable to connect to '" + url + "', " + err.message + " for " + ptw.projectId);
                sendSuccess = false;
            }

            if (!sendSuccess) {
                await backoffUtil.sleepAsync();
                backoffUtil.failIncrease();
            } else {
                backoffUtil.successReset();
            }
        }

    }

    public dispose() {
        if (this._disposed) { return; }

        log.info("dispose() called on FileWatcher");

        this._disposed = true;

        this._getStatusThread.dispose();
        this._watchService.dispose();
        this._outputQueue.dispose();
        this._webSocketManager.dispose();

        this._projectsMap.forEach( (e) => {
            e.batchUtil.dispose();
        });

    }

    public generateDebugString(): string {
        let result = "";

        if (this._disposed) {
            return "";
        }

        result += "---------------------------------------------------------------------------------------\n\n";

        result += "WatchService - " + this._watchService.constructor.name + ":\n";
        result += this._watchService.generateDebugState().trim() + "\n";

        result += "\n";

        result += "Project list:\n";

        for (const [key, value] of this._projectsMap) {

            const ptw = value.projectToWatch;

            result += "- " + key + " | " + ptw.pathToMonitor;

            if (ptw.ignoredPaths.length > 0) {
                result += " | ignoredPaths: ";
                for (const path of  ptw.ignoredPaths) {
                    result += "'" + path + "' ";
                }
            }
            result += "\n";
        }

        result += "\nHTTP Post Output Queue:\n" + this._outputQueue.generateDebugString().trim() + "\n\n";

        result += "---------------------------------------------------------------------------------------\n\n";

        return result;
    }

    private removeSingleProjectToWatch(removedProject: ProjectToWatch) {

        const po = this._projectsMap.get(removedProject.projectId);

        if (!po) {
            log.error("Asked to remove a project that wasn't in the projects map: " + removedProject.projectId);
            return;
        }

        const ptw = po.projectToWatch;

        log.info("Removing project from watch list: " + ptw.projectId + " " + ptw.pathToMonitor);
        this._projectsMap.delete(removedProject.projectId);

        const fileToMonitor = PathUtils.convertAbsoluteUnixStyleNormalizedPathToLocalFile(ptw.pathToMonitor);
        log.debug("Calling watch service removePath with file: " + fileToMonitor);

        this._watchService.removePath(fileToMonitor, ptw);

    }

    private createOrUpdateProjectToWatch(ptw: ProjectToWatch) {
        if (this._disposed) { return; }

        let po = this._projectsMap.get(ptw.projectId);

        const fileToMonitor = PathUtils.convertAbsoluteUnixStyleNormalizedPathToLocalFile(ptw.pathToMonitor);

        if (po === undefined) {
            po = new ProjectObject(ptw.projectId, ptw, this);
            this._projectsMap.set(ptw.projectId, po);

            this._watchService.addPath(fileToMonitor, ptw);

            log.info("Added new project with path '" + ptw.pathToMonitor
                + "' to watch list, with watch directory: '" + fileToMonitor + "'");

        } else {

            const oldProjectToWatch = po.projectToWatch;

            // If the watch has changed, then remove the path and update the PTw
            if (oldProjectToWatch.projectWatchStateId !== ptw.projectWatchStateId) {

                log.info("The project watch state has changed: " + oldProjectToWatch.projectWatchStateId + " "
                    + ptw.projectWatchStateId + " for project " + ptw.projectId);

                // Existing project to watch
                po.updateProjectToWatch(ptw);

                // Remove the old path
                this._watchService.removePath(fileToMonitor, oldProjectToWatch);

                log.info("From update, removed project with path '" + oldProjectToWatch.pathToMonitor
                    + "' from watch list, with watch directory: '" + fileToMonitor + "'");

                // Added the new path and PTW
                this._watchService.addPath(fileToMonitor, ptw);

                log.info("From update, added new project with path '" + ptw.pathToMonitor
                    + "' to watch list, with watch directory: '" + fileToMonitor + "'");

            } else {
                log.info("The project watch state has not changed for project " + ptw.projectId);
            }

        }

    }

}
