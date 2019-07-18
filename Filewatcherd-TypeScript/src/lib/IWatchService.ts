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
import { ProjectToWatch } from "./ProjectToWatch";

/**
 * This interface is an abstraction over platform-specific mechanisms for
 * detecting file/directory changes of a directory, in a lightweight fashion. On
 * Linux, this uses the inotify API, and Windows/Mac have similar
 * platform-specific APIs.
 *
 * In Node, we use the chokidar file change notification detection library (at present), but
 * one may implement this IWatchService interface to hook into the
 * IDE-specific mechanisms for detecting class changes.
 *
 * See WatchService for an example implementation.
 */
export interface IWatchService {

    /**
     * Add a path (and its subdirectories) to the list of directories to monitor.
     * After this call, any changes to files under this path should call the
     * IPlatformWatchListener listeners to be called.
     *
     * Method is called by FileWatcher to inform the watch service of a new
     * directory to watch.
     */
    addPath(fileToMonitor: string, ptw: ProjectToWatch): void;

    /**
     * Remove a path that was previously added with above path; ends monitoring on
     * the directory (and its subdirectories). The watch service should no longer
     * report any events under this path after this call.
     *
     * Method is called by FileWatcher to inform the watch service that a watched
     * directory should no longer be watched.
     */
    removePath(fileToMonitor: string, oldProjectToWatch: ProjectToWatch): void;

    setParent(parent: FileWatcher): void;

    /**
     * Dispose of any file watchers that are associated with this watch service, and
     * any other associated resources (threads, buffers, etc).
     *
     * Should be called by the watch service consumer when the Codewind connection
     * is disposed (for example, if the workbench is stopping, or the user has
     * deleted the server entry).
     */
    dispose(): void;

    /** Generate an implementation-defined String representation of the internal state of the watcher. */
    generateDebugState(): string;

}
