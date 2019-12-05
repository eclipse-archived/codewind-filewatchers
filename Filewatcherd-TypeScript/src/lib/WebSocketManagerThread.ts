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

import { clearTimeout } from "timers";
import * as WebSocket from "ws";
import { ExponentialBackoffUtil } from "./ExponentialBackoffUtil";
import { FileWatcher } from "./FileWatcher";

import * as models from "./Models";
import { ProjectToWatchFromWebSocket } from "./ProjectToWatchFromWebSocket";

import * as log from "./Logger";

/**
 * The purpose of this class is to initiate and maintain the WebSocket
 * connection between the filewatcher and the server.
 *
 * After queueEstablishConnection(...) is called, we will keep trying to connect
 * to the server until it succeeds. If that connection ever goes down for any
 * reason, queueEstablishConnection() still start the reconnection process over
 * again.
 *
 * This class also sends a simple "keep alive" packet every X seconds (eg 25).
 */
export class WebSocketManagerThread {

    private static readonly SEND_KEEPALIVE_EVERY_X_SECONDS = 25;

    private readonly _wsBaseUrl: string;

    private readonly _parent: FileWatcher;

    /** Maintains a reference to the previous websocket, to ensure it is closed when we open a new one */
    private _previousWebSocket: WebSocket;

    private _previousWebSocketInterval: NodeJS.Timer;

    private _attemptingToEstablish = false;

    private _disposed: boolean = false;

    constructor(wsBaseUrl: string, parent: FileWatcher) {
        this._wsBaseUrl = wsBaseUrl;
        this._parent = parent;

        this._previousWebSocket = null;
        this._previousWebSocketInterval = null;
    }

    public queueEstablishConnection() {
        if (this._disposed) { return; }

        if (!this._attemptingToEstablish) {
            this.establishOrReestablishConnectionAsync();
            log.info("INFO - Establish connection queued for '" + this._wsBaseUrl + "', and accepted.");
        } else {
            log.info("INFO - Establish connection queued for '" + this._wsBaseUrl + "', but ignored.");
        }
    }

    public dispose() {
        if (this._disposed) { return; }

        log.info("dispose() called on WebSocketManagerThread");

        this._disposed = true;

        this.disposeWebSocketAsync();
    }

    private async disposeWebSocketAsync() {
        try {
            if (this._previousWebSocket) {
                this._previousWebSocket.close();
            }
        } catch (e) {
            /* ignore*/
        }
    }

    /**
     * If the websocket connection fails, we should issue a new GET request to
     * ensure we have the latest state.
     */
    private refreshWatchStatus() {
        if (this._disposed) { return; }

        this._parent.refreshWatchStatus();
    }

    private async establishOrReestablishConnectionAsync() {
        if (this._disposed) { return; }

        if (this._attemptingToEstablish) {
            // If another thread is already in the middle of attempting to establish, then we shouldn't do
            // it in parallel.
            return;
        }

        // TODO: Add token-based authentication to the WebSocket, once support is implemented server-side.
        // (https://github.com/eclipse/codewind/issues/1342)

        try {

            if (this._previousWebSocket) {
                try { this._previousWebSocket.close(); } catch (e) { /* ignore*/ }
                this._previousWebSocket = null;
            }

            if (this._previousWebSocketInterval) {
                clearInterval(this._previousWebSocketInterval);
                this._previousWebSocketInterval = null;
            }

            this._attemptingToEstablish = true;

            let success = false;

            const delay = ExponentialBackoffUtil.getDefaultBackoffUtil(4000);

            let attemptNumber = 1;

            let ws: WebSocket = null;

            while (!success && !this._disposed) {
                log.info("Attempting to establish connection to web socket, attempt #" + attemptNumber);

                let timer = null;

                try {

                    ws = new WebSocket(this._wsBaseUrl + "/websockets/file-changes/v1", {
                        rejectUnauthorized: false,
                    });

                    const threadReference = this;

                    const promise = new Promise<boolean>((resolve, _) => {

                        timer = setTimeout(() => {
                            // Timeout after 15 seconds of waiting to open
                            resolve(false);
                        }, 15 * 1000);

                        ws.on("open", function open() {
                            log.debug("Connected to WebSocket, resolving promise.");
                            resolve(true);
                        });

                        ws.on("error", function error(en: WebSocket, errorFromFunction: Error) {
                            log.error("Error handler called on WebSocket", errorFromFunction);
                            resolve(false);
                        });

                        ws.on("message", function incoming(data) {
                            const str = data.toString();
                            log.debug("Message received from WebSocket: " + str);

                            try {
                                threadReference.receiveMessage(str);
                            } catch (e) {
                                log.severe("Exception caught by receiveMessage", e);
                            }
                        });

                        ws.on("close", function close(code: number, reason: string) {
                            log.error("WebSocket connection closed: " + code + " " + reason);
                            if (success) {
                                threadReference.queueEstablishConnection();
                                threadReference.refreshWatchStatus();
                            } else {
                                resolve(false);
                            }
                        });

                    });

                    success = await promise;

                } catch (e) {
                    log.error("Web socket error", e);
                    success = false;
                }

                if (timer) {
                    clearTimeout(timer);
                }

                if (success) {
                    log.info("Established connection to web socket, after attempt #" + attemptNumber);
                    this.refreshWatchStatus();
                } else {
                    log.error("Unable to establish connection to web socket on attempt #" + attemptNumber);
                    attemptNumber++;
                    if (!this._disposed) {
                        await delay.sleepAsync();
                        delay.failIncrease();
                    }
                }

            } // end while

            if (success) {
                this._previousWebSocket = ws;
                this._previousWebSocketInterval = setInterval(() => {
                    if (this._disposed) { return; }

                    if (ws.readyState === ws.OPEN) {
                        try { ws.send("{}"); } catch (e) { /* ignore*/ }
                    }
                }, WebSocketManagerThread.SEND_KEEPALIVE_EVERY_X_SECONDS * 1000);

            }

        } finally {
            this._attemptingToEstablish = false;
        }
    }
    private receiveMessage(s: string) {
        if (this._disposed) { return; }

        const fromJSON = JSON.parse(s);
        if (fromJSON.type === "debug") {
            this.handleDebug(fromJSON);
            return;
        }

        log.info("Received watch change message from WebSocket: " + s);

        const wc: IWatchChangeJson = JSON.parse(s);

        if (!wc || !wc.type || !wc.projects) {
            log.severe("Received invalid json string: " + s);
        }

        const projects: ProjectToWatchFromWebSocket[] = new Array<ProjectToWatchFromWebSocket>();

        // Output list of changes (before calling update on parent)
        {
            let infoStr = "";

            for (const e of wc.projects) {
                const ptw: ProjectToWatchFromWebSocket = ProjectToWatchFromWebSocket.create(e);
                projects.push(ptw);
                infoStr += "[" + ptw.projectId + " in " + (ptw.pathToMonitor ? ptw.pathToMonitor : "N/A") + "], ";
            }
            infoStr = infoStr.trim();
            infoStr = infoStr.substring(0, infoStr.length - 1); // strip the trailing comma

            log.info("Watch list update received for { " + infoStr + " }");
        }

        this._parent.updateFileWatchStateFromWebSocket(projects);

    }

    private handleDebug(debugMsg: any) {
        try {
            const msg = debugMsg.msg;
            log.info("------------------------------------------------------------");
            log.info("[Server-Debug] " + msg);
            log.info("------------------------------------------------------------");
        } catch (e) {
            /* ignore */
        }
    }
}

export interface IWatchChangeJson {
    projects: models.IWatchedProjectJson[];
    type: string;
}
