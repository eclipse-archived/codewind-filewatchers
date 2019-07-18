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

import * as models from "./Models";

import * as log from "./Logger";

import { ExponentialBackoffUtil } from "./ExponentialBackoffUtil";
import { FileWatcher } from "./FileWatcher";
import { ProjectToWatch } from "./ProjectToWatch";

/**
 * This class is responsible for issuing a GET request to the server in order to
 * retrieve the latest list of projects to watch (including their path, and any
 * filters).
 *
 * A new GET request will be sent by this class on startup, and after startup,
 * a GET request will be sent either:
 * - whenever the WebSocket connection fails
 * - or otherwise, once every X seconds (eg 120)
 *
 * WebSocketManagerThread is responsible for informing this class when the
 * WebSocket connection fails (input), and this class calls the Filewatcher
 * class with the data from the GET request (containing any project watch
 * updates received) as output.
 */
export class HttpGetStatusThread {

    public static readonly REFRESH_EVERY_X_SECONDS = 120;

    private _baseUrl: string;

    private _inInnerLoop: boolean;

    private _timer: NodeJS.Timer = null;

    private _disposed: boolean = false;

    private readonly _parent: FileWatcher;

    public constructor(baseUrl: string, parent: FileWatcher) {
        this._baseUrl = baseUrl;
        this._parent = parent;

        this.resetTimer();
        this.queueStatusUpdate();
    }

    public async queueStatusUpdate(): Promise<void> {
        if (this._disposed) { return; }
        if (this._inInnerLoop) { return; }

        this.innerLoop();

    }
    public dispose() {
        if (this._disposed) { return; }

        log.info("dispose() called on HttpGetStatusThread");

        this._disposed = true;
    }

    private async doHttpGet(): Promise<ProjectToWatch[]> {

        const options = {
            json: true,
            resolveWithFullResponse: true,
            timeout: 20000,
        };

        try {

            const url = this._baseUrl + "/api/v1/projects/watchlist";

            log.info("Initiating GET request to " + url);

            const httpResult = await request.get(url, options);

            if (httpResult.statusCode && httpResult.statusCode === 200 && httpResult.body ) {

                // Strip EOL characters to ensure it fits on one log line.
                let bodyVal = JSON.stringify(httpResult.body);
                bodyVal = bodyVal.replace("\n", "");
                bodyVal = bodyVal.replace("\r", "");

                log.info("GET response received: " + bodyVal);

                const w: models.IWatchedProjectListJson = httpResult.body;
                if (w == null || w === undefined) {
                    log.error("Expected value not found for GET watchlist endpoint");
                    return null;
                }

                const result = new Array<ProjectToWatch>();

                for (const e of w.projects) {

                    // Santity check the json parsing
                    if (!e.projectID || !e.pathToMonitor) {
                        log.error("JSON parsing of GET watchlist endpoint failed with missing values");
                        return null;
                    }

                    result.push(new ProjectToWatch(e, false));
                }

                return result;
            } else {
                return null;
            }

        } catch (err) {
            log.error("GET request failed. [" + (err.message) + "]");

            return null;
        }

    }
    private async innerLoop(): Promise<void> {
        try {
            this._inInnerLoop = true;
            log.debug("HttpGetStatus - beginning GET request loop.");

            let success = false;

            const delay = ExponentialBackoffUtil.getDefaultBackoffUtil(4000);

            let  result: ProjectToWatch[] = null;

            while (!success && !this._disposed) {

                result = await this.doHttpGet();

                success = result != null;

                if (!success) {
                    log.error("HTTP get request failed");
                    await delay.sleepAsync();
                    delay.failIncrease();
                }

            }

            if (result && result.length > 0) {
                this._parent.updateFileWatchStateFromGetRequest(result);
            }

        } finally {
            log.info("HttpGetStatus - GET request loop complete.");
            this._inInnerLoop = false;
            this.resetTimer();
        }

    }
    private resetTimer() {
        if (this._timer != null) {
            clearTimeout(this._timer);
        }

        // Refresh every X seconds (eg 120)
        this._timer = setTimeout(() => {
            this.queueStatusUpdate();
        }, HttpGetStatusThread.REFRESH_EVERY_X_SECONDS * 1000);

    }

}
