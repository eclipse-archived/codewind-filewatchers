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

import * as chokidar from "chokidar";
import { WatchEventEntry } from "./WatchEventEntry";
import * as watchEvents from "./WatchEventEntry";
import { WatchService } from "./WatchService";

import * as log from "./Logger";

import { PathFilter } from "./PathFilter";
import { ProjectToWatch } from "./ProjectToWatch";

import { existsSync, readdir, statSync } from "fs";

import pathLib = require("path");
import { convertAbsolutePathWithUnixSeparatorsToProjectRelativePath, normalizePath } from "./PathUtils";

/**
 * The entry in WatchService for an individual directory to (recursively) watch. There
 * should be a 1-1 relationship between WatchedPath objects and projects that are
 * monitored on behalf of the server.
 *
 * Change events are first detected via the 3rd party Chokidar library. We takes those
 * events, do some initial processing, then pass them to watch service (which will forward
 * them on FileWatcher).
 *
 * This class also handles the case where we are asked to watch a directory that doesn't
 * yet exist. We will wait up to 5 minutes for a directory to exist, before we report an error
 * back to the server.
 */
export class WatchedPath {

    private readonly _pathRoot: string;

    private readonly _normalizedPath: string;

    private _watcher: chokidar.FSWatcher;

    private readonly _parent: WatchService;

    private _watchIsReady: boolean;

    private _projectToWatch: ProjectToWatch;

    private _pathFilter: PathFilter;

    private _disposed: boolean = false;

    public constructor(pathRoot: string, parent: WatchService, projectToWatch: ProjectToWatch) {

        this._parent = parent;
        this._pathRoot = pathRoot;
        this._normalizedPath = normalizePath(pathRoot);

        this._pathFilter = new PathFilter(projectToWatch);

        this._projectToWatch = projectToWatch;

        this._watchIsReady = false;

        this.waitForPathToExist(0, new Date());
    }

    public waitForPathToExist(attempts: number, startTime: Date) {

        let success = false;

        try {
            if (existsSync(this._pathRoot) && statSync(this._pathRoot).isDirectory()) {
                success = true;
                this.init();
            }
        } catch (e) { /* ignore*/ }

        if (!success) {
            if (attempts % 5 === 0) {
                log.info("Waiting for path to exist: " + this._pathRoot);
            }

            if (new Date().getTime() - startTime.getTime() > 60 * 5 * 1000) {
                // After 5 minutes, stop waiting for the watch and report the error
                log.error("Watch failed on " + this._pathRoot);
                this._parent.parent.sendWatchResponseAsync(false, this._projectToWatch);

            } else {
                // Wait up to 5 minutes for the directory to appear.
                setTimeout(() => {
                    this.waitForPathToExist(attempts + 1, startTime);
                }, 1000);

            }

        }
    }

    public init() {
        const normalizedPathConst = this._normalizedPath;
        const pathFilterConst = this._pathFilter;

        /** Decide what to ignore using our custom function; true if should ignore, false otherwise. */
        const ignoreFunction = (a: string, _: any) => {
            if (!a) { return false; }

            a = convertAbsolutePathWithUnixSeparatorsToProjectRelativePath(normalizePath(a), normalizedPathConst);

            if (!a) { return false; }

            return pathFilterConst.isFilteredOutByFilename(a) || pathFilterConst.isFilteredOutByPath(a);
        };

        // Use polling on Windows, due to high CPU and memory usage of non-polling when using chokidar.
        // Use polling on Mac due to unreliable directory deletion detection.
        const usePollingVal = true; // os.platform().indexOf("win32") !== -1 || os.platform().indexOf("darwin") !== -1;

        this._watcher = chokidar.watch(this._pathRoot, {
            binaryInterval: 2000,
            disableGlobbing: true,
            ignoreInitial: true,
            ignored: ignoreFunction,
            interval: 250,
            persistent: true,
            usePolling: usePollingVal,
        });

        this._watcher
            .on("add", (path) => this.handleFileChange(path, "CREATE"))
            .on("change", (path) => this.handleFileChange(path, "MODIFY"))
            .on("unlink", (path) => this.handleFileChange(path, "DELETE"));

        this._watcher
            .on("addDir", (path) => this.handleDirChange(path, "CREATE"))
            .on("unlinkDir", (path) => this.handleDirChange(path, "DELETE"))
            .on("error", (error) => log.severe(`Watcher error: ${error}`, error))
            .on("raw", (event, path, _) => {
                log.debug("Watcher raw event info:" + event + "|" + path);

                if (path === this._pathRoot && !existsSync(path)) {
                    this.handleDirChange(path, "DELETE");
                }

            })
            .on("ready", () => {
                if (this._disposed) { return; }
                this._watchIsReady = true;
                log.info("Initial scan of '" + this._pathRoot + "' complete. Ready for changes");
                this._parent.parent.sendWatchResponseAsync(true, this._projectToWatch);
            });

        // this._watcher.on("change", (path, stats) => {
        //     if (stats) { console.log(`File ${path} changed size to ${stats.size}`); }
        // });
    }

    public dispose() {

        if (this._disposed) { return; }
        this._disposed = true;

        log.info("dispose() called on WatchedPath for " + this._pathRoot);

        this._watchIsReady = false;

        this.closeWatcherAsync();

    }

    private async closeWatcherAsync() {

        // As of Chokidar 3.2.0, we should only close the watch if the directory exists; if the directory
        // doesn't exist then Chokidar has already cleaned it up on its own.
        if (existsSync(this._pathRoot) && this._watcher) {
            this._watcher.close();
        }
    }

    private handleFileChange(path: string, type: string) {

        if (this._disposed) { return; }

        // File changes are not valid until the watch is ready.
        if (!this._watchIsReady) {
            log.debug("Ignoring  file " + path);
            return;
        }

        const eventType = watchEvents.getEventTypeFromString(type);

        const entry = new WatchEventEntry(eventType, path, false);

        this._parent.handleEvent(entry);
    }

    private handleDirChange(path: string, type: string) {

        if (this._disposed) { return; }

        // File changes are not valid until the watch is ready.
        if (!this._watchIsReady) {
            log.debug("Ignoring dir " + path);
            return;
        }

        const eventType = watchEvents.getEventTypeFromString(type);

        const entry = new WatchEventEntry(eventType, path, true);

        this._parent.handleEvent(entry);

        // If it is the root directory that is going away, then dispose of the watcher.
        if (path === this._pathRoot && type === "DELETE") {
            log.debug("Watcher observed deletion of the root path " + this._pathRoot);

            this.dispose();
        }

    }

    public get pathRoot(): string {
        return this._pathRoot;
    }

    /**
     * Recursively scan a directory and create a list of any files/folders found inside. My thinking
     * in creating this was it would reduce incidents of events being missed by Chokidar, but the code
     * had no obvious effect.
     * @param path Path to scan
     */
    private scanCreatedDirectory(path: string) {
        const directoriesFound: string[] = [];
        const filesFound: string[] = [];

        const newWatchEvents = new Array<WatchEventEntry>();

        this.scanCreatedDirectoryInner(path, directoriesFound, filesFound);

        for (const dir of directoriesFound) {
            newWatchEvents.push(new WatchEventEntry(watchEvents.EventType.CREATE, dir, true));
        }

        for (const file of filesFound) {
            newWatchEvents.push(new WatchEventEntry(watchEvents.EventType.CREATE, file, false));
        }

        for (const we of newWatchEvents) {
            this._parent.handleEvent(we);
        }
    }

    private scanCreatedDirectoryInner(path: string, directoriesFound: string[], filesFound: string[]) {
        readdir(path, (err, files) => {

            if (err) {
                log.error("Error received when attempting to read directory: " + path, err);
                return;
            }

            if (!files) { return; }

            files.forEach((fileInner) => {
                if (!fileInner) { return; }

                const fullPath = path + pathLib.sep + fileInner;

                if (statSync(fullPath).isDirectory()) {
                    if (fullPath === path) { return; }
                    directoriesFound.push(fullPath);
                    this.scanCreatedDirectoryInner(fullPath, directoriesFound, filesFound);
                } else {
                    filesFound.push(fullPath);
                }
            });
        });
    }
}
