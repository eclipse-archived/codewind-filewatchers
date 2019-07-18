import { PathFilter } from "./PathFilter";
import * as pathutils from "./PathUtils";
import { ProjectToWatch } from "./ProjectToWatch";
import { VSCodeResourceWatchService } from "./VSCodeResourceWatchService";
import { WatchEventEntry } from "./WatchEventEntry";

/**
 * The entry in VSCodeResourceWatchService for an individual directory to
 * (recursively) watch. There should be a 1-1 relationship between WatchedPath
 * objects and projects that are monitored on behalf of the server.
 *
 * The receiveFileChanges(...) method is called by the parent, from which we
 * process and filter the change, then pass it to filewatcher core through the
 * IPlatformWatchListener interface.
 */
export class VSCWatchedPath {

    private readonly _pathInNormalizedForm: string;

    private readonly _pathFilter: PathFilter;

    private readonly _parent: VSCodeResourceWatchService;

    private readonly _pathRoot: string;

    constructor(pathRoot: string, ptw: ProjectToWatch, parent: VSCodeResourceWatchService) {

        this._pathInNormalizedForm = pathutils.normalizePath(pathRoot);

        this._pathFilter = new PathFilter(ptw);

        this._parent = parent;

        this._pathRoot = pathRoot;

        this._parent.parent.sendWatchResponseAsync(true, ptw);

    }
    public receiveFileChanges(entries: WatchEventEntry[]) {

        const newEvents: WatchEventEntry[] = [];

        for (const wee of entries) {
            const relativePath = pathutils.convertAbsolutePathWithUnixSeparatorsToProjectRelativePath(
                wee.absolutePathWithUnixSeparators, this._pathInNormalizedForm);

            if (!relativePath || (!this._pathFilter.isFilteredOutByFilename(relativePath)
                && !this._pathFilter.isFilteredOutByPath(relativePath)) ) {
                    newEvents.push(wee);
                }
        }

        if (newEvents.length > 0) {

            for (const event of newEvents) {
                this._parent.internal_handleEvent(event);
            }
        }
    }

    public dispose() {
        /* Nothing to dispose */
    }

    public get pathRoot(): string {
        return this._pathRoot;
    }

}
