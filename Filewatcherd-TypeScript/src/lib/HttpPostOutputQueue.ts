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

import * as request from "request-promise-native";
import { ExponentialBackoffUtil } from "./ExponentialBackoffUtil";

import * as log from "./Logger";
import { PostQueueChunk } from "./PostQueueChunk";
import { PostQueueChunkGroup } from "./PostQueueChunkGroup";

export class HttpPostOutputQueue {
    private static readonly MAX_ACTIVE_REQUESTS = 3;

    private readonly _queue: PostQueueChunkGroup[];

    private readonly _serverBaseUrl: string;

    private _activeRequests: number;

    private _disposed: boolean = false;

    private readonly _failureDelay: ExponentialBackoffUtil;

    constructor(serverBaseUrl: string) {
        this._serverBaseUrl = serverBaseUrl;
        this._queue = new Array<PostQueueChunkGroup>();
        this._activeRequests = 0;
        this._failureDelay = ExponentialBackoffUtil.getDefaultBackoffUtil(4000);
    }

    // TODO: Remove OutputQueueEntry when HttpPostOutputQueueOld is removed.

    public async informStateChangeAsync() {
        // While there is more work, and we are below request capacity
        while (this._activeRequests < HttpPostOutputQueue.MAX_ACTIVE_REQUESTS && !this._disposed) {

            const chunk: PostQueueChunk = this.getNextPieceOfWork();
            if (chunk == null) {
                // No work available, so return
                return;
            }

            try {
                this._activeRequests++;

                await this.sendRequestAsync(chunk);

            } finally {

                this._activeRequests--;

            }
        }
    }

    public addToQueue(projectId: string, timestamp: number, base64Compressed: string[]) {
        if (this._disposed) { return; }

        log.info("addToQueue called with " + base64Compressed.length + " entries.");

        const chunkGroup = new PostQueueChunkGroup(timestamp, projectId, base64Compressed, this);

        this._queue.push(chunkGroup);

        this.resortQueue();
        this.informStateChangeAsync();
    }
    public dispose() {
        if (this._disposed) { return; }

        log.info("dispose() called on HttpPostOutputQueue");

        this._disposed = true;
    }

    public generateDebugString(): string {
        let result = "- ";

        if (this._disposed) {
            return result + "[disposed]";
        }

        result += "total-workers: " + HttpPostOutputQueue.MAX_ACTIVE_REQUESTS
            + "  active-workers: " + this._activeRequests;

        result += "  chunkGroupList-size: " + this._queue.length + "\n";

        if (this._queue.length > 0) {
            result += "\n";
            result += "- HTTP Post Chunk Group List:\n";

            for (const chunkGroup of this._queue) {
                result += "  - projectID: " + chunkGroup.projectId + "  timestamp: " + chunkGroup.timestamp + "\n";
            }

        }

        return result;
    }

    /** Remove any chunk groups that have already sent all their chunks. */
    private cleanupChunkGroups(): void {
        let changeMade = false;

        for (let i = this._queue.length - 1; i >= 0; i--) {
            if (this._queue[i].isGroupComplete()) {
                this._queue.splice(i, 1);
                changeMade = true;
            }
        }
        if (changeMade) {
            this.informStateChangeAsync();
        }
    }

    /** Returns next available piece of work from the queue, or null if no work available. */
    private getNextPieceOfWork(): PostQueueChunk {

        this.cleanupChunkGroups();
        if (this._queue.length === 0) { return null; }

        const group: PostQueueChunkGroup = this._queue[0];
        const nextChunk = group.acquireNextChunkAvailableToSend();
        return nextChunk;

    }

    private async doHttpPost(url: string, chunk: PostQueueChunk): Promise<boolean> {

        if (this._disposed) { return false; }

        const payload: any = {
            msg: chunk.base64Compressed,
        };

        const options = {
            body: payload,
            json: true,
            resolveWithFullResponse: true,
            timeout: 20000,
        };

        log.info("Issuing POST request to '" + url + "'");
        try {
            const result = await request.post(url, options);

            if (result.statusCode !== 200) {
                log.error("Unexpected error code " + result.statusCode + " from '" + url + "'");
                return false;
            }

            log.debug("POST request to '" + url + "' succeeded.");

            return true;

        } catch (err) {
            log.error("Unable to connect to '" + url + "', " + err.message);
        }
        return false;

    }

    private resortQueue() {

        // Sort ascending by timestamp
        this._queue.sort((a, b) => {
            return a.timestamp - b.timestamp;
        });
    }

    private async sendRequestAsync(chunk: PostQueueChunk) {

        const url = this._serverBaseUrl + "/api/v1/projects/" + chunk.projectId + "/file-changes?timestamp="
            + chunk.timestamp + "&chunk=" + chunk.chunkId + "&chunk_total=" + chunk.chunkTotal;

        const postResult: boolean = await this.doHttpPost(url, chunk);

        if (postResult) {
            this._failureDelay.successReset();
            chunk.parentGroup.informPacketSent(chunk);
        } else {

            if (this._disposed) { return; }

            // On failure, delay adding back to the queue to rate limit the POST operations.
            await this._failureDelay.sleepAsync();

            // Increase the length of the failure delay after each failure (to a maximum of
            // 4000 msecs); on success, reduce the failure back to default. (retry w/
            // exponential backoff)
            this._failureDelay.failIncrease();

            chunk.parentGroup.informPacketFailedToSend(chunk);

        }

    }

}
