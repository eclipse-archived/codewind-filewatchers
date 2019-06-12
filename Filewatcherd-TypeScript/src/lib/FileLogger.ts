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

import fs = require("fs");
import os = require("os");
import path = require("path");
import { LogSettings } from "./Logger";
import * as log from "./Logger";

export class FileLogger {

    public static readonly FILE_PREFIX = "filewatcherd-";

    public static readonly FILE_SUFFIX = ".log";

    public static readonly MAX_LOG_FILE_SIZE = 1024 * 1024 * 12;

    private _logDir: string;

    private _bytesWritten = 0;

    private _nextNumber = 1;

    private _fd: number = -1;

    private _initialized: boolean = false;

    private readonly _parent: LogSettings;

    public constructor(logDir: string) {
        this._logDir = logDir;

        this.log("codewind-filewatcher logging to " + logDir
            + " with log level " + log.logLevelToString(LogSettings.getInstance().logLevel));
    }

    public log(str: string) {
        try {
            this.logInner(str);
        } catch (e) {
            /* ignore */
        }
    }

    private logInner(str: string) {

        if (this._fd === -2) { return; }

        if (!this._initialized) {
            if (fs.existsSync(this._logDir)) {
                if (!this._initialized) {
                    this._initialized  = true;

                    const list = fs.readdirSync(this._logDir);
                    for (const val of list) {

                        if (val.startsWith(FileLogger.FILE_PREFIX) && val.endsWith(FileLogger.FILE_SUFFIX)) {
                            fs.unlinkSync(path.join(this._logDir, val));
                        }
                    }
                }

            } else {
                return;
            }

        }

        if (this._fd < 0 || this._bytesWritten > FileLogger.MAX_LOG_FILE_SIZE) {
            if (this._fd === -2) { return; }

            if (this._fd >= 0) {
                fs.close(this._fd, (e) => {
                    console.error("Unable to close file descriptor.");
                });
            }

            const oldPath = path.join(this._logDir, FileLogger.FILE_PREFIX
                + (this._nextNumber - 2) + FileLogger.FILE_SUFFIX);

            // Delete if the oldPath exists
            fs.exists( oldPath, (exists) => {
                if (!exists) { return; }
                fs.unlink(oldPath, (err) => {
                    console.error("Unable to delete " + oldPath + " " + err);
                });
            });

            this._bytesWritten = 0;

            const pathVar = path.join(this._logDir, FileLogger.FILE_PREFIX + this._nextNumber + FileLogger.FILE_SUFFIX);
            this._nextNumber++;

            this._fd = -2;

            this._fd = fs.openSync(pathVar, "a");
        }

        if (this._fd >= 0) {
            fs.appendFileSync(this._fd, str + os.EOL, "utf8");
            this._bytesWritten += str.length;
        }

    }
}
