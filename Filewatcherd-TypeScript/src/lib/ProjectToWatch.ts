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
// import { ProjectToWatchFromWebSocket } from "./ProjectToWatchFromWebSocket";

/**
 * Contains information on what directory to (recursively monitor), and any
 * filters that should be applied (eg to ignore changes to files/directories with
 * specific names/paths).
 *
 * The fields correspond to the array of projects JSON from
 * the 'GET api/v1/projects/watchlist' API. See docs for format restrictions for
 * these fields.
 */
export class ProjectToWatch {

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

    public get projectCreationTimeInAbsoluteMsecs(): number {
        return this._projectCreationTimeInAbsoluteMsecs;
    }

    /** Create an instance of this class from the given JSON object. */
    public static createFromJson(json: models.IWatchedProjectJson, deleteChangeType: boolean): ProjectToWatch {
        const result = new ProjectToWatch();
        ProjectToWatch.innerCreateFromJson(result, json, deleteChangeType);
        return result;
    }

    /**
     * Create a new ProjectToWatch (not ProjectToWatchFromWebSocket), copy the values from the old param,
     * but replace the projectCreationTimeInAbsoluteMsecs param
     */
    public static cloneWithNewProjectCreationTime(old: ProjectToWatch,
                                                  projectCreationTimeInAbsoluteMsecsParam: number): ProjectToWatch {

        // if (old instanceof ProjectToWatchFromWebSocket) {
        //     // Sanity test
        //     throw new Error("cloneWithNewProjectCreationTime should not be called with a FromWebSocket object");
        // }
        const result = new ProjectToWatch();

        ProjectToWatch.copyWithNewProjectCreationTime(result, old, projectCreationTimeInAbsoluteMsecsParam);

        return result;
    }

    /**
     * Copy values from old to result, but replace projectCreationTimeInAbsoluteMsecsParam param in result.
     * Called only by above method, and from ProjectToWatchFromWebSocket.
     */
    protected static copyWithNewProjectCreationTime(result: ProjectToWatch, old: ProjectToWatch,
                                                    projectCreationTimeInAbsoluteMsecsParam: number) {

        result._external = old.external;

        result._projectId = old.projectId;
        result._pathToMonitor = old.pathToMonitor;

        result._projectWatchStateId = old.projectWatchStateId;

        result.validatePathToMonitor();

        const ignoredPaths: string[] = [];
        if (old.ignoredPaths && old.ignoredPaths.length > 0) {
            old.ignoredPaths.forEach((e) => { ignoredPaths.push(e); });
        }
        result._ignoredPaths = ignoredPaths;

        const ignoredFilenames: string[] = [];
        if (old.ignoredFilenames && old.ignoredFilenames.length > 0) {
            old.ignoredFilenames.forEach((e) => { ignoredFilenames.push(e); });
        }
        result._ignoredFilenames = ignoredFilenames;

        // Replace the old value, with specified parameter.
        result._projectCreationTimeInAbsoluteMsecs = projectCreationTimeInAbsoluteMsecsParam;

    }

    /** Copy the values from the JSON object into the given ProjectToWatch. */
    protected static innerCreateFromJson(result: ProjectToWatch, json: models.IWatchedProjectJson,
                                         deleteChangeType: boolean) {

        // Delete event from WebSocket only has these fields.
        if (deleteChangeType) {
            result._projectId = json.projectID;
            result._pathToMonitor = null;
            result._projectWatchStateId = null;
            return;
        }

        result._projectId = json.projectID;

        result._pathToMonitor = PathUtils.normalizeDriveLetter(json.pathToMonitor);

        result.validatePathToMonitor();

        const ignoredPaths: string[] = [];
        if (json.ignoredPaths && json.ignoredPaths.length > 0) {
            json.ignoredPaths.forEach((e) => { ignoredPaths.push(e); });
        }
        result._ignoredPaths = ignoredPaths;

        const ignoredFilenames: string[] = [];
        if (json.ignoredFilenames && json.ignoredFilenames.length > 0) {
            json.ignoredFilenames.forEach((e) => { ignoredFilenames.push(e); });
        }
        result._ignoredFilenames = ignoredFilenames;

        result._projectWatchStateId = json.projectWatchStateId;

        result._external = json.type ? (json.type.toLowerCase() === "non-project") : false;

        result._projectCreationTimeInAbsoluteMsecs = json.projectCreationTime;

    }

    /**
     * The contents of this class are defacto immutable after creation, and should not be changed.
     * However, the fields are not read-only because we need multiple constructing methods.
     */

    private _projectId: string;
    private _pathToMonitor: string;

    private _ignoredPaths: string[];
    private _ignoredFilenames: string[];

    private _projectWatchStateId: string;

    private _external: boolean;

    /** undefined if project time is not specified, a >0 value otherwise. */
    private _projectCreationTimeInAbsoluteMsecs: number;

    protected constructor() { }

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
}
