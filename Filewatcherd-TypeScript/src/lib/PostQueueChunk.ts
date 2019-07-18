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

import { PostQueueChunkGroup } from "./PostQueueChunkGroup";

/**
 * A large number of file changes will be split into 'bite-sized pieces' called
 * chunks. Each chunk communicates a subset of the full change list.
 *
 * Instances of this class are immutable.
 */
export class PostQueueChunk {
    private readonly _projectId: string;
    private readonly _timestamp: number;
    private readonly _base64Compressed: string;

    /** The ID of a chunk will be 1 <= id <= chunkTotal */
    private readonly _chunkId: number;

    /** The total # of chunks that will e sent for this project id and timestamp. */
    private readonly _chunkTotal: number;

    private readonly _parentGroup: PostQueueChunkGroup;

    constructor(projectId: string, timestamp: number, base64Compressed: string, chunkId: number, chunkTotal: number,
                parentGroup: PostQueueChunkGroup) {

        this._projectId = projectId;
        this._timestamp = timestamp;
        this._base64Compressed = base64Compressed;
        this._chunkId = chunkId;
        this._chunkTotal = chunkTotal;
        this._parentGroup = parentGroup;
    }

    public get projectId(): string {
        return this._projectId;
    }

    public get timestamp(): number {
        return this._timestamp;
    }

    public get base64Compressed(): string {
        return this._base64Compressed;
    }

    public get chunkId(): number {
        return this._chunkId;
    }

    public get chunkTotal(): number {
        return this._chunkTotal;
    }

    public get parentGroup(): PostQueueChunkGroup {
        return this._parentGroup;
    }
}
