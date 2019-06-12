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

export class OutputQueueEntry {
    private readonly _timestamp: number;

    private readonly _messageToSend: string;

    private readonly _projectId: string;

    constructor(projectId: string, timestamp: number, messageToSend: string) {
        this._projectId = projectId;
        this._timestamp = timestamp;
        this._messageToSend = messageToSend;

    }

    public get timestamp(): number {
        return this._timestamp;
    }

    public get messageToSend(): string {
        return this._messageToSend;
    }

    public get projectId(): string {
        return this._projectId;
    }

}
