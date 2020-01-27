/*******************************************************************************
* Copyright (c) 2020 IBM Corporation and others.
* All rights reserved. This program and the accompanying materials
* are made available under the terms of the Eclipse Public License v2.0
* which accompanies this distribution, and is available at
* http://www.eclipse.org/legal/epl-v20.html
*
* Contributors:
*     IBM Corporation - initial API and implementation
*******************************************************************************/

/** Struct class containing the most recent observed state of a file we have been told to watch. */
export class PollEntry {
    public lastObservedStatus: PollEntryStatus = PollEntryStatus.RECENTLY_ADDED;

    public absolutePath: string;

    // 0 if the file doesn't exist, or if the status is RECENTLY_ADDED
    public lastModifiedDate: number = 0;

    constructor(lastObservedStatus: PollEntryStatus , absolutePath: string , lastModifiedDate: number) {
        this.lastObservedStatus = lastObservedStatus;
        this.absolutePath = absolutePath;
        this.lastModifiedDate = lastModifiedDate;
    }
}

export enum PollEntryStatus {
    /**
     * We were recently told to watch this file and thus have not yet observed a
     * state for it.
     */
    RECENTLY_ADDED,

    /** File exists, last time we checked it. */
    EXISTS,

    /** File did not exist, last time we checked it. */
    DOES_NOT_EXIST,
}
