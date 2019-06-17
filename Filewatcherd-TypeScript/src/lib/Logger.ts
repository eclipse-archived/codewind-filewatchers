
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

import { FileLogger } from "./FileLogger";

export enum LogLevel {
    DEBUG = 1,
    INFO,
    ERROR,
    SEVERE,
}

export class LogSettings {

    public get logLevel(): LogLevel {
        return this._logLevel;
    }

    public set logLevel(e: LogLevel) {
        this._logLevel = e;
    }

    public static getInstance(): LogSettings {
        return LogSettings.instance;
    }

    private static instance: LogSettings = new LogSettings();

    private _logLevel: LogLevel;

    private _startTimeInMsecs: number;

    private _fileLogger: FileLogger = null;

    private _outputLogsToScreen: boolean = true;

    constructor() {
        this._logLevel = LogLevel.INFO;
        this._startTimeInMsecs = new Date().getTime();
    }

    public generatePrefix(): string {
        const elapsed = new Date().getTime() - this._startTimeInMsecs;

        const seconds = Math.floor(elapsed / 1000);
        const msecs = 1000 + (elapsed % 1000);

        const msecsString = ("" + msecs).substring(1);

        let properDate = "";
        {
            const date = new Date();

            const milliseconds = 1000 + date.getMilliseconds();

            // request a weekday along with a long date
            const options = {
                day: "numeric",
                formatMatcher : "basic",
                hour: "numeric",
                hour12 : false,
                minute: "2-digit",
                month: "long",
                second: "2-digit",
                // weekday: "short",
                // year: "numeric",
            };
            properDate = "[" + new Intl.DateTimeFormat("en-US", options).format(date)
                + "." + milliseconds.toString().substring(1, 4) + "]";

        }

        return properDate + " [" + seconds + "." + msecsString + "] ";
    }

    public setOutputLogsToScreen(outputToScreen: boolean) {
        this._outputLogsToScreen = outputToScreen;
    }

    public setFileLogger(fileLogger: FileLogger) {
        this._fileLogger = fileLogger;
    }

    public internalGetFileLogger(): FileLogger {
        return this._fileLogger;
    }

    public get outputLogsToScreen(): boolean {
        return this._outputLogsToScreen;
    }

}

export function setLogLevel(l: LogLevel) {
    LogSettings.getInstance().logLevel = l;
}

export function debug(str: string) {
    if (LogSettings.getInstance().logLevel > LogLevel.DEBUG) {
        return;
    }

    const prefix = LogSettings.getInstance().generatePrefix();

    const msg = prefix + str;
    printOut(msg);

}

export function info(str: string) {
    if (LogSettings.getInstance().logLevel > LogLevel.INFO) {
        return;
    }

    const prefix = LogSettings.getInstance().generatePrefix();

    const msg = prefix + str;

    printOut(msg);

}

export function error(str: string, err?: Error) {
    if (LogSettings.getInstance().logLevel > LogLevel.ERROR) {
        return;
    }

    const prefix = LogSettings.getInstance().generatePrefix();

    const suffix = err ? ": " : "";

    const msg = prefix + "! ERROR !  " + str + suffix;

    printErr(msg);
    if (err) {
        printErr(err);
    }

}

export function severe(str: string, err?: Error) {

    const prefix = LogSettings.getInstance().generatePrefix();

    const suffix = err ? ": " : "";

    const msg = prefix + "!!! SEVERE !!!  " + str + suffix;

    printErr(msg);
    if (err) {
        printErr(err);
    }
}

function printOut(entry: any) {

    const outputLogsToScreen = LogSettings.getInstance().outputLogsToScreen;

    const fileLogger = LogSettings.getInstance().internalGetFileLogger();
    if (fileLogger != null) {
        fileLogger.log(entry);
    }

    if (outputLogsToScreen) {
        console.log(entry);
    }

}

function printErr(entry: any) {
    const fileLogger = LogSettings.getInstance().internalGetFileLogger();

    const outputLogsToScreen = LogSettings.getInstance().outputLogsToScreen;

    if (entry instanceof Error) {
        const entryError: Error = entry;
        const msg = entryError.name + " " + entryError.message + " " + entryError.stack;

        if (outputLogsToScreen) {
            console.error(msg);
        }
        if (fileLogger != null) {
            fileLogger.log(msg);
        }

    } else {
        // Non-error
        if (outputLogsToScreen) {
            console.error(entry);
        }
        if (fileLogger != null) {
            fileLogger.log(entry);
        }

    }

}

export function logLevelToString(l: LogLevel) {
    if (l === LogLevel.DEBUG) {
        return "DEBUG";
    } else if (l === LogLevel.ERROR) {
        return "ERROR";
    } else if (l === LogLevel.INFO) {
        return "INFO";
    } else {
        return "SEVERE";
    }
}
