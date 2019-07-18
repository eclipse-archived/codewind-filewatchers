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

import * as we from "./WatchEventEntry";

/**
 * Simple representation of a single change: the file/dir path that changed,
 * what type of change, and when. These are then consumed by the batch
 * processing utility.
 */
export class ChangedFileEntry {
    private readonly _eventType: we.EventType;

    private readonly _timestamp: number;
    private readonly _path: string;

    private readonly _directory: boolean;

    constructor(path: string, type: we.EventType, timestamp: number, directory: boolean) {
        if (!path || !type || timestamp < 0) {
            throw new Error("Invalid parameter '" + path + "' '" + type + "' '" + timestamp + "'");
        }

        this._eventType = type;
        this._path = path;
        this._timestamp = timestamp;
        this._directory = directory;
    }

    public toJson(): IChangedFileEntryJson {

        const result: IChangedFileEntryJson = {
            directory : this._directory,
            path  : this._path,
            timestamp : this._timestamp,
            type : we.eventTypetoString(this._eventType),
        };

        return result;
    }

    public get timestamp(): number {
        return this._timestamp;
    }

    public get eventType(): we.EventType {
        return this._eventType;
    }

    public get path(): string {
        return this._path;
    }

    public get directory(): boolean {
        return this._directory;
    }
}

/** Represents a single file change that is then communicated and sent in the HTTP POST request. */
export interface IChangedFileEntryJson {
    path: string;
    timestamp: number;
    type: string;
    directory: boolean;
}
