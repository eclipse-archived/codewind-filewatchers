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

import * as log from "./Logger";
import * as PathUtils from "./PathUtils";

export enum EventType {
    CREATE = 1,
    MODIFY,
    DELETE,
}

export function eventTypetoString(type: EventType): string {
    if (type === EventType.CREATE) {
        return "CREATE";
    } else if (type === EventType.MODIFY) {
        return "MODIFY";
    } else if (type === EventType.DELETE) {
        return "DELETE";
    } else {
        const msg = "Unable to convert event type to string - " + type;
        log.severe(msg);
        throw new Error(msg);
    }

}

export function getEventTypeFromString(str: string): EventType {
    if (str === "CREATE") {
        return EventType.CREATE;
    } else if (str === "MODIFY") {
        return EventType.MODIFY;
    } else if (str === "DELETE") {
        return EventType.DELETE;
    } else {
        const msg = "Unable to find event type: this shouldn't happen - " + str;
        log.severe(msg);
        throw new Error(msg);
    }
}

/**
 * Corresponds to an file/directory change event from the OS: a
 * creation/modification/deletion of a file, or a creation/deletion of a
 * directory.
 */
export class WatchEventEntry {

    private readonly _path: string;

    private readonly _eventType: EventType;

    private readonly _directory: boolean;

    constructor(eventType: EventType, pathParam: string, directory: boolean) {
        this._eventType = eventType;

        this._path = PathUtils.normalizePath(pathParam);
        this._directory = directory;
    }

    /** For debugging only */
    public toString(): string {
        return "[" + eventTypetoString(this._eventType) + "] " + this._path;
    }

    public get absolutePathWithUnixSeparators(): string {
        return this._path;
    }

    public get eventType(): EventType {
        return this._eventType;
    }

    public get directory(): boolean {
        return this._directory;
    }

}
