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
import { OutputQueueEntry } from "./OutputQueueEntry";

import * as log from "./Logger";

export class HttpPostOutputQueueNew {

    private static readonly MAX_ACTIVE_REQUESTS = 3;

    private readonly _queue: OutputQueueEntry[];

    private readonly _serverBaseUrl: string;

    private _activeRequests: number;

    private _disposed: boolean = false;

    private readonly _failureDelay: ExponentialBackoffUtil;

    constructor(serverBaseUrl: string) {
        this._serverBaseUrl = serverBaseUrl;
        this._queue = new Array<OutputQueueEntry>();
        this._activeRequests = 0;
        this._failureDelay = ExponentialBackoffUtil.getDefaultBackoffUtil(4000);
    }

    public addToQueue(projectId: string, timestamp: number, base64Compressed: string[]) {
        if (this._disposed) { return; }

        log.info("addToQueue called with " + base64Compressed.length + " entries.");

        for (const e of base64Compressed) {
            this._queue.push(new OutputQueueEntry(projectId, timestamp, e));
        }

        this.resortQueue();
        this.queueIfNeededAsync();
    }
    public dispose() {
        if (this._disposed) { return; }

        log.info("dispose() called on HttpPostOutputQueue");

        this._disposed = true;
    }

    public addToQueueFromRetry(entryToRetry: OutputQueueEntry) {
        if (this._disposed) { return; }

        log.info("Added file changes to queue, for retry");

        this._queue.push(entryToRetry);

        this.resortQueue();
        this.queueIfNeededAsync();

    }

    private async doHttpPost(url: string, oqe: OutputQueueEntry): Promise<boolean> {

        if (this._disposed) { return false; }

        const payload: any = {
            msg: oqe.messageToSend,
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

            log.info("POST request to '" + url + "' succeeded.");

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

    private async queueIfNeededAsync() {

        // While there is more work, and we are below request capacity
        while (this._activeRequests < HttpPostOutputQueueNew.MAX_ACTIVE_REQUESTS && this._queue.length > 0
            && !this._disposed) {

            try {
                this._activeRequests++;

                await this.pollAndSend();

            } finally {

                this._activeRequests--;

            }
        }
    }

    private async pollAndSend() {

        if (this._queue.length === 0) { return; }

        const oqe = this._queue.splice(0, 1)[0];

        if (!oqe) { return; }

        const url = this._serverBaseUrl + "/api/v1/projects/" + oqe.projectId + "/file-changes?timestamp="
            + oqe.timestamp;

        const postResult: boolean = await this.doHttpPost(url, oqe);

        if (postResult) {
            this._failureDelay.successReset();
        } else {

            if (this._disposed) { return; }

            // On failure, delay adding back to the queue to rate limit the POST operations.
            await this._failureDelay.sleepAsync();

            // Increase the length of the failure delay after each failure (to a maximum of
            // 4000 msecs); on success, reduce the failure back to default. (retry w/
            // exponential backoff)
            this._failureDelay.failIncrease();

            this.addToQueueFromRetry(oqe);
        }

    }

}
