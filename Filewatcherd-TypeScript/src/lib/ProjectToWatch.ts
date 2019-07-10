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

import * as models from "./Models";
import * as PathUtils from "./PathUtils";

export class ProjectToWatch {

    /**
     * The contents of this class are immutable after creation, and should not be changed.
     * Do not add non-readonly fields to this class.
     */

    private readonly _projectId: string;
    private readonly _pathToMonitor: string;

    private readonly _ignoredPaths: string[];
    private readonly _ignoredFilenames: string[];

    private readonly _projectWatchStateId: string;

    private readonly _external: boolean;

    constructor(json: models.IWatchedProjectJson, deleteChangeType: boolean) {

        // Delete event from WebSocket only has these fields.
        if (deleteChangeType) {
            this._projectId = json.projectID;
            this._pathToMonitor = null;
            this._projectWatchStateId = null;
            return;
        }

        this._projectId = json.projectID;

        this._pathToMonitor = PathUtils.normalizeDriveLetter(json.pathToMonitor);

        this.validatePathToMonitor();

        const ignoredPaths: string[] = [];
        if (json.ignoredPaths && json.ignoredPaths.length > 0) {
            json.ignoredPaths.forEach((e) => { ignoredPaths.push(e); });
        }
        this._ignoredPaths = ignoredPaths;

        const ignoredFilenames: string[] = [];
        if (json.ignoredFilenames && json.ignoredFilenames.length > 0) {
            json.ignoredFilenames.forEach((e) => { ignoredFilenames.push(e); });
        }
        this._ignoredFilenames = ignoredFilenames;

        this._projectWatchStateId = json.projectWatchStateId;

        this._external = json.type ? (json.type.toLowerCase() === "non-project") : false;
    }

    private validatePathToMonitor() {

        if (this._pathToMonitor.indexOf("\\") !== -1) {
            throw new Error(
                "Path to monitor should not contain Windows-style path separators: " + this._pathToMonitor);
        }

        if (!this._pathToMonitor.startsWith("/")) {
            throw new Error(
                "Path to monitor should always begin with a forward slash: " + this._pathToMonitor);
        }

        if (this._pathToMonitor.endsWith("/") || this._pathToMonitor.endsWith("\\")) {
            throw new Error(
                "Path to monitor may not end with path separator: " + this._pathToMonitor);
        }
    }

    public get projectId(): string {
        return this._projectId;
    }

    public get pathToMonitor(): string {
        return this._pathToMonitor;
    }

    public get ignoredPaths(): string[] {
        return this._ignoredPaths;
    }

    public get ignoredFilenames(): string[] {
        return this._ignoredFilenames;
    }

    public get projectWatchStateId(): string {
        return this._projectWatchStateId;
    }

    public get external(): boolean {
        return this._external;
    }
}
