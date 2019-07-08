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

import { FileWatcher } from "./FileWatcher";
import { IWatchService } from "./IWatchService";
import * as log from "./Logger";
import { ProjectToWatch } from "./ProjectToWatch";
import { VSCWatchedPath } from "./VSCWatchedPath";
import { WatchEventEntry } from "./WatchEventEntry";

export class VSCodeResourceWatchService implements IWatchService {

    private readonly _projIdToWatchedPaths = new Map<string /* project Id to Watched Path */, VSCWatchedPath>();

    private _parent: FileWatcher;

    private _disposed: boolean = false;

    constructor() {
        /* empty */
    }

    public addPath(fileToMonitor: string, ptw: ProjectToWatch): void {
        if (this._disposed) { return; }

        log.info("Path '" + fileToMonitor + "' added to watcher for " + ptw.projectId );

        const key = ptw.projectId;

        const value = this._projIdToWatchedPaths.get(key);

        if (value) {
            value.dispose();
        }

        this._projIdToWatchedPaths.set(key, new VSCWatchedPath(fileToMonitor, ptw, this));

    }

    public removePath(fileToMonitor: string, oldProjectToWatch: ProjectToWatch): void {
        if (this._disposed) { return; }

        const key = oldProjectToWatch.projectId;

        const value = this._projIdToWatchedPaths.get(key);
        if (value) {
            log.info("Path '" + fileToMonitor + "' removed from " + key );
            value.dispose();
        } else {
            log.error("Path '" + fileToMonitor + "' attempted to be removed, but could not be found in " + key);
        }

    }

    public receiveWatchEntries(cwProjectId: string, entries: WatchEventEntry[]) {

        const wp: VSCWatchedPath  = this._projIdToWatchedPaths.get(cwProjectId);
        if (wp) {
            wp.receiveFileChanges(entries);
        } else {
            // TODO: If this is printed for projects that are not managed by Codewind, then just comment this out.
            log.error("Could not find project with ID '" + cwProjectId + "' in list.");
        }
    }

    public internal_handleEvent(event: WatchEventEntry) {
        if (this._disposed) { return; }

        try {
            log.info("Received event " + event.toString());

            const timeReceived = Math.round(new Date().getTime());
            this._parent.receiveNewWatchEventEntry(event, timeReceived);
        } catch (e) {
            log.severe("handleEvent caught an uncaught exception.", e);
        }
    }

    public setParent(parent: FileWatcher): void {
        this._parent = parent;
    }

    public dispose(): void {
        this._disposed = true;
        this._projIdToWatchedPaths.clear();

        log.info("Dispose called.");

        for (const e of this._projIdToWatchedPaths.values()) {
            e.dispose();
        }

    }

    public generateDebugState(): string {
        if (this._disposed) { return "[disposed]"; }

        let result = "";

        for (const [key, value] of this._projIdToWatchedPaths) {
            result += "- " + key + " | " + value.pathRoot + "\n";
        }

        return result;
    }

}
