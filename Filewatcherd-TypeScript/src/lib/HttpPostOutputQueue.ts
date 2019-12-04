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

import { AuthTokenWrapper } from "./AuthTokenWrapper";
import * as log from "./Logger";
import { PostQueueChunk } from "./PostQueueChunk";
import { PostQueueChunkGroup } from "./PostQueueChunkGroup";

/**
 * This class is responsible for informing the server (via HTTP POST request) of
 * any file/directory changes that have occurred.
 *
 * The FileChangeEventBatchUtil (indirectly) calls this class with a list of
 * base-64+compressed strings (containing the list of changes), and then this
 * class breaks the changes down into small chunks and sends them in the body of
 * individual HTTP POST requests.
 */
export class HttpPostOutputQueue {
    private static readonly MAX_ACTIVE_REQUESTS = 3;

    /**
     * After X hours (eg 24), give up on trying to send this chunk group to the
     * server. At this point the data is too stale to be useful.
     */
    private static readonly CHUNK_GROUP_EXPIRE_TIME_IN_MSECS = 1000 * 60 * 60 * 24;

    /**
     * On other platforms we use a priority queue here, sorted ascending by timestamp; here we use a list, and just sort
     * it every time we add to it, to achieve the same goal.
     */
    private readonly _queue: PostQueueChunkGroup[];

    private readonly _serverBaseUrl: string;

    /** Number of I/O threads that are currently busy attempting to send HTTP POST requests */
    private _activeRequests: number;

    private _disposed: boolean = false;

    private readonly _failureDelay: ExponentialBackoffUtil;

    private readonly _authTokenWrapper: AuthTokenWrapper;

    constructor(serverBaseUrl: string, authTokenWrapper: AuthTokenWrapper) {
        this._serverBaseUrl = serverBaseUrl;
        this._queue = new Array<PostQueueChunkGroup>();
        this._activeRequests = 0;
        this._failureDelay = ExponentialBackoffUtil.getDefaultBackoffUtil(4000);
        this._authTokenWrapper = authTokenWrapper;
    }

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

        const chunkGroup = new PostQueueChunkGroup(timestamp, projectId,
            new Date().getTime() + HttpPostOutputQueue.CHUNK_GROUP_EXPIRE_TIME_IN_MSECS, base64Compressed, this);

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

    /**
     * Remove any chunk groups that have already sent all their chunks, or that have
     * expired (unable to send communication for X hours, eg 24)
     */
    private cleanupChunkGroups(): void {
        let changeMade = false;

        const currentTime = new Date().getTime();

        for (let i = this._queue.length - 1; i >= 0; i--) {

            const chunkGroup = this._queue[i];

            if (chunkGroup.isGroupComplete()) {
                this._queue.splice(i, 1);
                changeMade = true;
            } else if (currentTime > chunkGroup.expireTime) {
                this._queue.splice(i, 1);
                changeMade = true;

                log.severe("Chunk group expired. This implies we could not connect to server for many hours."
                    + " Chunk-group project: " + chunkGroup.projectId + "  timestamp: " + chunkGroup.timestamp);
            }
        }
        if (changeMade) {
            // Inform threads waiting for work
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
            followRedirect: false,
            json: true,
            rejectUnauthorized : false,
            resolveWithFullResponse: true,
            timeout: 20000,
        } as request.RequestPromiseOptions;

        const authToken = this._authTokenWrapper.getLatestToken();
        if (authToken && authToken.accessToken) {

            options.auth = {
                bearer: authToken.accessToken,
            };
        }

        log.info("Issuing POST request to '" + url + "'");
        try {
            const result = await request.post(url, options);

            if (result.statusCode !== 200) {

                // TODO: In the unlikely event that this class is NOT deleted when cwctl is stable, then convert
                // this to check for 302, like the other HTTP endpoints
                if (authToken && authToken.accessToken && result.statusCode && result.statusCode === 403) {
                    this._authTokenWrapper.informBadToken(authToken);
                }

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
