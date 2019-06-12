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

import zlib = require("zlib");
import { ChangedFileEntry, IChangedFileEntryJson } from "./ChangedFileEntry";
import { FileWatcher } from "./FileWatcher";
import * as log from "./Logger";
import { EventType } from "./WatchEventEntry";

export class FileChangeEventBatchUtil {

    private static readonly TIME_TO_WAIT_FOR_NO_NEW_EVENTS_IN_MSECS = 1000;

    private static readonly MAX_REQUEST_SIZE_IN_PATHS = 625;

    private _files: ChangedFileEntry[];

    private _timer: NodeJS.Timer = null;

    private _disposed: boolean = false;

    private readonly _parent: FileWatcher;

    private readonly _projectId: string;

    public constructor(projectId: string, parent: FileWatcher) {
        this._parent = parent;
        this._files = new Array<ChangedFileEntry>();
        this._projectId = projectId;
    }

    /**
     * When files have changed, add them to the list and reset the timer task ahead
     * X millseconds.
     */
    public addChangedFiles(changedFileEntries: ChangedFileEntry[]) {
        if (this._disposed) { return; }

        for (const entry of changedFileEntries) {
            this._files.push(entry);
        }

        if (this._timer != null) {
            clearTimeout(this._timer);
        }

        this._timer = setTimeout(() => {
            try {
                this.doTimerTask();
            } catch (e) {
                log.severe("TimerTask failed", e);
            }
        }, FileChangeEventBatchUtil.TIME_TO_WAIT_FOR_NO_NEW_EVENTS_IN_MSECS);
    }
    public dispose() {
        if (this._disposed) { return; }

        log.info("dispose() called on FileChangeEventBatchUtil");

        this._disposed = true;
    }

    private doTimerTask() {
        if (this._disposed) { return; }

        let entries: ChangedFileEntry[] = [];

        for (const entry of this._files) {
            entries.push(entry);
        }
        this._files = [];

        // Clear the timeout if it already exists.
        clearTimeout(this._timer);
        this._timer = null;

        if (entries.length === 0) {
            return;
        }

        // Sort ascending (JGW: confirmed as ascending)
        entries = entries.sort((n1, n2) => {
            const val1 = n1.timestamp;
            const val2 = n2.timestamp;
            if (val1 > val2) {
                return 1;
            } else if (val2 > val1) {
                return -1;

            } else {
                return 0;
            }

        });

        const mostRecentTimestamp = entries[entries.length - 1];

        const eventSummary = this.generateChangeListSummaryForDebug(entries);
        log.info("Batch change summary for " + this._projectId + " @ "
            + mostRecentTimestamp.timestamp + ": " + eventSummary);

        // Split the entries into requests, ensure that each request is no larger
        // then a given size.
        const fileListsToSend = new Array<IChangedFileEntryJson[]>();
        while (entries.length > 0) {
            const currList: IChangedFileEntryJson[] = new Array<IChangedFileEntryJson>();
            while (currList.length < FileChangeEventBatchUtil.MAX_REQUEST_SIZE_IN_PATHS && entries.length > 0) {
                const nextPath = entries.splice(0, 1);

                currList.push(nextPath[0].toJson());
            }

            if (currList.length > 0) {
                fileListsToSend.push(currList);
            }
        }

        const base64Compressed = new Array<string>();
        for (const array of fileListsToSend) {
            const str = JSON.stringify(array);
            // log.debug("JSON contents: " + str);
            const strBuffer = zlib.deflateSync(str);
            base64Compressed.push(strBuffer.toString("base64"));
        }

        if (base64Compressed.length > 0) {
            this._parent.sendBulkFileChanges(this._projectId, mostRecentTimestamp.timestamp, base64Compressed);
        }
    }

    /* Output the first 256 characters of the change list, as a summary of the full list of
     * changes. This means the change list is not necessary a complete list, and is only
     * what fits into the given length. */
    private generateChangeListSummaryForDebug(entries: ChangedFileEntry[]): string {
        let result = "[ ";

        for (const entry of entries) {

            if (entry.eventType === EventType.CREATE) {
                result += "+";
            } else if (entry.eventType === EventType.MODIFY) {
                result += ">";
            } else if (entry.eventType === EventType.DELETE) {
                result += "-";
            } else {
                result += "?";
            }

            let filename = entry.path;
            const index = filename.lastIndexOf("/");
            if (index !== -1) {
                filename = filename.substring(index + 1);
            }
            result += filename + " ";

            if (result.length > 256) {
                break;
            }
        }

        if (result.length > 256) {
            result += " (...) ";
        }
        result += "]";

        return result;

    }
}
