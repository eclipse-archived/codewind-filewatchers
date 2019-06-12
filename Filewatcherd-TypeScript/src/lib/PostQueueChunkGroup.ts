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

import { HttpPostOutputQueue } from "./HttpPostOutputQueue";
import { PostQueueChunk } from "./PostQueueChunk";

import * as log from "./Logger";

export enum ChunkStatus { AVAILABLE_TO_SEND, WAITING_FOR_ACK, COMPLETE }

export class PostQueueChunkGroup {
    private readonly _chunkMap: Map<number, PostQueueChunk> = new Map();

    private readonly _chunkStatus: Map<number, ChunkStatus> = new Map();

    private readonly _parent: HttpPostOutputQueue;

    private readonly _timestamp: number;

    constructor(timestamp: number, projectId: string, base64Compressed: string[], parent: HttpPostOutputQueue) {
        this._parent = parent;
        this._timestamp = timestamp;

        let chunkId = 1;

        for (const message of base64Compressed) {
            const chunk = new PostQueueChunk(projectId, timestamp, message, chunkId, base64Compressed.length, this);

            this._chunkMap.set(chunk.chunkId, chunk);
            this._chunkStatus.set(chunk.chunkId, ChunkStatus.AVAILABLE_TO_SEND);

            chunkId++;
        }

    }

    /** A group is complete if every chunk is ChunkStatus.COMPLETE */
    public isGroupComplete(): boolean {
        for (const chunkStatus of this._chunkStatus.values()) {
            if (chunkStatus !== ChunkStatus.COMPLETE) {
                return false;
            }
        }

        return true;
    }

    public informPacketSent(chunk: PostQueueChunk): void {

        const val = this._chunkStatus.get(chunk.chunkId);
        if (!val || val !== ChunkStatus.WAITING_FOR_ACK) {
            log.severe("Unexpected status of chunk, should be WAITING, but was:" + val);
            return;
        }

        // Set the chunk back to complete, so no one else sends it
        this._chunkStatus.set(chunk.chunkId, ChunkStatus.COMPLETE);

        this._parent.informStateChangeAsync();
    }

    public informPacketFailedToSend(chunk: PostQueueChunk): void {

        const val = this._chunkStatus.get(chunk.chunkId);
        if (!val || val !== ChunkStatus.WAITING_FOR_ACK) {
            log.severe("Unexpected status of chunk, should be WAITING, but was:" + val);
            return;
        }

        // Reset the chunk back to AVAILABLE_TO_SEND, so someone else can send it
        this._chunkStatus.set(chunk.chunkId, ChunkStatus.AVAILABLE_TO_SEND);

        this._parent.informStateChangeAsync();
    }

    /** Returns the next chunk to be sent, or empty if none are currently available. */
    public acquireNextChunkAvailableToSend(): PostQueueChunk {

        let matchingEntry = null;

        for (const mapEntry of this._chunkStatus) {

            const [key, value] = mapEntry;
            if (value === ChunkStatus.AVAILABLE_TO_SEND) {
                matchingEntry = key;
            }

        }

        if (matchingEntry == null) {
            return null;
        }

        this._chunkStatus.set(matchingEntry, ChunkStatus.WAITING_FOR_ACK);

        return this._chunkMap.get(matchingEntry);

    }

    public get timestamp(): number {
        return this._timestamp;
    }

}
