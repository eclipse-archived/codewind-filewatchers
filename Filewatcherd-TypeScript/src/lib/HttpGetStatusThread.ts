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

import got from "got";

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

        const requestObj = {
            header: {},
            rejectUnauthorized: false,
            retry: 0,
            timeout: 20000,
        };

        const authTokenWrapper = this._parent.authTokenWrapper;
        const authToken = authTokenWrapper.getLatestToken();
        if (authToken && authToken.accessToken) {
            requestObj.header = { bearer: authToken.accessToken };
        }

        try {

            const url = this._baseUrl + "/api/v1/projects/watchlist";

            log.info("Initiating GET request to " + url);

            const response = await got(url, requestObj);

            if (response.statusCode  === 200 && response.body) {

                // Strip EOL characters to ensure it fits on one log line.
                const w = JSON.parse(response.body);

                const debugVal = JSON.stringify(w).replace("\n", "").replace("\r", "");

                log.info("GET response received: " + debugVal);

                if (w == null || w === undefined) {
                    log.error("Expected value not found for GET watchlist endpoint");
                    return null;
                }

                const result = new Array<ProjectToWatch>();

                for (const e of w.projects) {

                    // Sanity check the JSON parsing
                    if (!e.projectID || !e.pathToMonitor) {
                        log.error("JSON parsing of GET watchlist endpoint failed with missing values");
                        return null;
                    }

                    result.push(ProjectToWatch.createFromJson(e, false));
                }

                return result;

            } else {

                if (response) {
                    log.info("GET request failed, details: " + response.statusCode + " " + response.body);
                }

                // Inform bad token if we are redirected to an OIDC endpoint
                if (authToken && authToken.accessToken && response.statusCode && response.statusCode === 302
                    && response.headers && response.headers.location
                    && response.headers.location.indexOf("openid-connect/auth") !== -1) {

                    authTokenWrapper.informBadToken(authToken);
                }

                return null;
            }

        } catch (err) {
            log.error("GET request failed. [" + (err.message) + "] (" + err.statusCode + ")");

            // Inform bad token if we are redirected to an OIDC endpoint
            if (err.statusCode === 302 && err.response && err.response.headers && err.response.headers.location
                && err.response.headers.location.indexOf("openid-connect/auth") !== -1) {

                if (authToken && authToken.accessToken) {
                    authTokenWrapper.informBadToken(authToken);
                }

            }

            return null;
        }

    }
    private async innerLoop(): Promise<void> {
        try {
            this._inInnerLoop = true;
            log.debug("HttpGetStatus - beginning GET request loop.");

            let success = false;

            const delay = ExponentialBackoffUtil.getDefaultBackoffUtil(4000);

            let result: ProjectToWatch[] = null;

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
