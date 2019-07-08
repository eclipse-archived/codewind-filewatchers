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
import { WatchedPath } from "./WatchedPath";
import { WatchEventEntry } from "./WatchEventEntry";

import * as log from "./Logger";
import { ProjectToWatch } from "./ProjectToWatch";

export class WatchService implements IWatchService {

    private _parent: FileWatcher;

    private readonly _watchedProjects: Map<string /* project id of watched project */, WatchedPath>;

    private _disposed: boolean = false;

    constructor() {
        this._watchedProjects = new Map<string, WatchedPath>();
    }

    public setParent(parent: FileWatcher) {
        this._parent = parent;
    }

    public addPath(fileToMonitor: string, ptw: ProjectToWatch) {
        if (this._disposed) { return; }

        const key = ptw.projectId;

        log.info("Path '" + fileToMonitor + "' added to WatchService for " + key);

        const wp  = this._watchedProjects.get(key);
        if (wp) {
            wp.dispose();
        }

        this._watchedProjects.set(key, new WatchedPath(fileToMonitor, this, ptw));
    }

    public removePath(fileToMonitor: string, oldProjectToWatch: ProjectToWatch) {
        if (this._disposed) { return; }

        const key = oldProjectToWatch.projectId;

        const wp = this._watchedProjects.get(key);

        if (wp) {
            log.info("Path '" + fileToMonitor + "' removed from WatchService for project " + key);
            wp.dispose();
            this._watchedProjects.delete(key);

        } else {
            log.error("Path '" + fileToMonitor + "' attempted to be removed for project "
                + key + " , but could not be found in WatchService.");
        }
    }

    public handleEvent(event: WatchEventEntry) {
        if (this._disposed) { return; }

        try {
            log.info("Received event " + event.toString());

            const timeReceived = Math.round(new Date().getTime());
            this._parent.receiveNewWatchEventEntry(event, timeReceived);
        } catch (e) {
            log.severe("handleEvent caught an uncaught exception.", e);
        }
    }

    public dispose() {
        if (this._disposed) { return; }

        log.info("dispose() called on WatchService");

        this._disposed = true;

        for (const [_, value] of this._watchedProjects) {
            value.dispose();
        }
    }

    public get parent() {
        return this._parent;
    }

    public generateDebugState(): string {
        if (this._disposed) { return "[disposed]"; }

        let result = "";

        for (const [key, value] of this._watchedProjects) {
            result += "- " + key + " | " + value.pathRoot + "\n";
        }

        return result;
    }

}
