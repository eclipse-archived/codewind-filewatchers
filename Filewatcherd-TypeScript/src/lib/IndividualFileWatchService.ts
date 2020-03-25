
/*******************************************************************************
* Copyright (c) 2020 IBM Corporation and others.
* All rights reserved. This program and the accompanying materials
* are made available under the terms of the Eclipse Public License v2.0
* which accompanies this distribution, and is available at
* http://www.eclipse.org/legal/epl-v20.html
*
* Contributors:
*     IBM Corporation - initial API and implementation
*******************************************************************************/

import fs = require("fs");

import { FileWatcher } from "./FileWatcher";
import {
    convertAbsoluteUnixStyleNormalizedPathToLocalFile,
} from "./PathUtils";

import { ChangedFileEntry } from "./ChangedFileEntry";
import * as log from "./Logger";
import { PollEntry } from "./PollEntry";
import { PollEntryStatus } from "./PollEntry";
import { EventType } from "./WatchEventEntry";

/**
 * This class is used to watch a small number of individual files, for example,
 * linked files defined in the 'refPaths' field of a watched project. For a
 * large number of files to watch, the watch service should be used instead.
 *
 * Files watched by this class do not need to exist
 *
 * A single instance of this class will exist per filewatcher (eg it is not per
 * project).
 *
 * This class was introduced as part of 'Project sync support for reference to
 * files outside of project folder' (codewind/1399).
 */
export class IndividualFileWatchService {

    private _filesToWatchMap: Map<string /* project id */, Map<string /* absolute path */, PollEntry
        /* linked files */>>;

    private _filewatcher: FileWatcher;

    private _disposed: boolean = false;

    private _timer: NodeJS.Timeout | undefined;

    constructor(filewatcher: FileWatcher) {
        this._filewatcher = filewatcher;
        this._filesToWatchMap = new Map<string, Map<string, PollEntry>>();

        this.threadRun();

    }

    public setFilesToWatch(projectId: string, pathsFromPtw: string[]) {

        const paths: string[] = [];
        for (const pathFromPtw of pathsFromPtw) {

            const newPath = convertAbsoluteUnixStyleNormalizedPathToLocalFile(pathFromPtw);

            const isDirectory = fs.existsSync(newPath) && fs.lstatSync(newPath).isDirectory();

            // Filter out and report directories
            if (isDirectory) {
                log.error("Project '" + projectId + "' was asked to watch a directory, which is not supported: "
                    + newPath);
                continue;
            }

            paths.push(newPath);
        }

        if (paths.length === 0) {
            return;
        }

        let mapUpdated = false;

        const currProjectState = this._filesToWatchMap.get(projectId);

        if (!currProjectState) {

            // This is a new project we haven't seen.

            const newFiles = new Map<string, PollEntry>();

            for (const path of paths) {
                const pollEntry = new PollEntry(PollEntryStatus.RECENTLY_ADDED, path, 0);
                newFiles.set(pollEntry.absolutePath, pollEntry);
                log.info("Files to watch - recently added for new project: " + path);
            }
            mapUpdated = true;

            this._filesToWatchMap.set(projectId, newFiles);
        } else {
            // This is an existing project with at least one file we are currently monitoring.

            for (const path of paths) {

                const pe = currProjectState.get(path);
                if (!pe) {

                    log.info("Files to watch - recently added for existing project: " + path);
                    currProjectState.set(path, new PollEntry(PollEntryStatus.RECENTLY_ADDED, path, 0));
                    mapUpdated = true;

                } else {
                    // Ignore: the path is in both maps -- no change.
                }
            }

            const pathsInParam = new Set<string>();
            for (const path of paths) {
                pathsInParam.add(path);
            }

            const keysToRemove = [];

            // Look for values that are in curr project state, but not in the parameter list. These are
            // files that we WERE watching, but are no longer.
            for (const [pathInCurrentState,] of currProjectState) {

                if (!pathsInParam.has(pathInCurrentState)) {
                    keysToRemove.push(pathInCurrentState);
                    log.info("Files to watch - removing from watch list: " + pathInCurrentState);
                    mapUpdated = true;
                }
            }
            for (const keyToRemove of keysToRemove) {
                currProjectState.delete(keyToRemove);
            }

            // If we're not watching anything anymore, remove the project from the state list.
            if (currProjectState.size === 0) {
                this._filesToWatchMap.delete(projectId);
            }

        } // end existing project else

    } // end method

    public dispose() {
        if (this._disposed) {
            return;
        }
        this._disposed = true;

        this._filesToWatchMap.clear();

    }

    private threadRun() {

        if (this._timer) {
            clearTimeout(this._timer);
        }

        this._timer = setTimeout(() => {
            try {
                this.innerThreadRun();
            } catch (e) {
                log.severe("TimerTask failed", e);
            }
            // Restart the timer on completion of thread.
            if (!this._disposed) {
                this.threadRun();
            }
        }, 2000);
    }

    private innerThreadRun() {
        const fileChangesDetected = new Map<string /* project id */, Set<ChangedFileEntry>>();

        const localMap = new Map<string /* project id */, PollEntry[]>();

        // Clone filesToWatchMap
        this._filesToWatchMap.forEach((watchFileState: Map<string, PollEntry>, projectId: string) => {

            const pollEntriesCopy = [];
            for (const pollEntry of watchFileState.values()) {
                pollEntriesCopy.push(pollEntry);
            }
            localMap.set(projectId, pollEntriesCopy);
        });

        // For each project we are monitoring...
        for (const [projectId, filesToWatch] of localMap) {
            // For each watched file in that project...
            for (const fileToWatch of filesToWatch) {
                const fileExists = fs.existsSync(fileToWatch.absolutePath);
                const newStatus = fileExists ? PollEntryStatus.EXISTS : PollEntryStatus.DOES_NOT_EXIST;

                let fileModifiedTime = 0;
                if (fileExists) {
                    const stats = fs.statSync(fileToWatch.absolutePath);
                    if (stats) {
                        fileModifiedTime = stats.mtime.getTime();
                    } else {
                        log.info("Unable to read file modified time: " + fileToWatch.absolutePath);
                    }
                }

                if (fileToWatch.lastObservedStatus !== PollEntryStatus.RECENTLY_ADDED) {

                    if (fileToWatch.lastObservedStatus !== newStatus) {

                        let type = EventType.CREATE;

                        if (fileExists) {
                            // ADDED: Last time we saw this file it did not exist, but now it does.
                            log.info("Watched file now exists: " + fileToWatch.absolutePath);
                            type = EventType.CREATE;
                        } else {
                            // DELETED: Last time we saw this file it did exist, but it no longer does.
                            log.info("Watched file has been deleted: " + fileToWatch.absolutePath);
                            type = EventType.DELETE;
                        }

                        let changedFiles = fileChangesDetected.get(projectId);
                        if (!changedFiles) {
                            changedFiles = new Set<ChangedFileEntry>();
                            fileChangesDetected.set(projectId, changedFiles);
                        }

                        changedFiles.add(new ChangedFileEntry(fileToWatch.absolutePath, type,
                            new Date().getTime(), false));
                    }

                    if (fileModifiedTime && fileModifiedTime > 0 && fileToWatch.lastModifiedDate
                        && fileToWatch.lastModifiedDate > 0 && fileModifiedTime !== fileToWatch.lastModifiedDate) {

                        // CHANGED: Last time we same this file it had a different modified time.
                        log.info("Watched file change detected: " + fileToWatch.absolutePath + " "
                            + fileModifiedTime + " " + fileToWatch.lastModifiedDate);

                        let changedFiles = fileChangesDetected.get(projectId);
                        if (!changedFiles) {
                            changedFiles = new Set<ChangedFileEntry>();
                            fileChangesDetected.set(projectId, changedFiles);
                        }

                        changedFiles.add(new ChangedFileEntry(fileToWatch.absolutePath, EventType.MODIFY,
                            new Date().getTime(), false));

                    }
                } // end if RECENTLY_ADDED block

                fileToWatch.lastObservedStatus = newStatus;
                fileToWatch.lastModifiedDate = fileModifiedTime;

            } // end filesToWatch iteration

        } // end localMap iteration

        for (const [projectId, paths] of fileChangesDetected) {
            if (paths.size === 0) {
                continue;
            }

            this._filewatcher.internal_receiveIndividualChangesFileList(projectId, paths);
        }

    } // end method
}
