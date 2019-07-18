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

import * as log from "./Logger";

/**
 * Various utilities related to converting Windows-style paths (ex: c:\Users)
 * to/from our standardized Unix-style format (ex: /C/users), in order to
 * conform to our watcher API specification.
 */

/**
 * A windows absolute path will begin with a letter followed by a colon: C:\
 */
function isWindowsAbsolutePath(absolutePath: string): boolean {
    if (absolutePath.length < 2) {
        return false;
    }

    const char0 = absolutePath.charAt(0);

    if (!isLetter(char0)) {
        return false;
    }

    if (absolutePath.charAt(1) !== ":") {
        return false;
    }

    return true;
}

/** Ensure that the drive is lowercase for Unix-style paths from Windows. */
export function normalizeDriveLetter(absolutePath: string): string {

    if (absolutePath.indexOf("\\") !== -1) {
        throw new Error("This function does not support Windows-style paths: " + absolutePath);
    }
    if (absolutePath.length < 2) {
        return absolutePath;
    }

    if (!absolutePath.startsWith("/")) {
        throw new Error("Path should begin with forward slash: " + absolutePath);
    }

    const char0 = absolutePath.charAt(0);
    const char1 = absolutePath.charAt(1);

    // Special case the absolute path of only 2 characters.
    if (absolutePath.length === 2) {
        if (char0 === "/" && isLetter(char1) && isUpperCase(char1)) {
            return "/" + char1.toLowerCase();
        } else {
            return absolutePath;
        }

    }

    const char2 = absolutePath.charAt(2);
    if (char0 === "/" && char2 === "/" && isLetter(char1) && isUpperCase(char1)) {

        return "/" + char1.toLowerCase() + char2 + absolutePath.substring(3);

    } else {
        return absolutePath;
    }
}

/** C:\helloThere -> /c/helloThere */
export function convertFromWindowsDriveLetter(absolutePath: string): string {

    if (!isWindowsAbsolutePath(absolutePath)) {
        return absolutePath;
    }

    // Replace \ with /
    absolutePath = convertBackSlashesToForwardSlashes(absolutePath);

    const char0 = absolutePath.charAt(0);

    // Strip first two characters
    absolutePath = absolutePath.substring(2);

    absolutePath = "/" + char0.toLowerCase() + absolutePath;

    return absolutePath;

}

/* Convert /c/Users/Administrator to c:\Users\Administrator */
function convertAbsoluteUnixStyleNormalizedPathToLocalFileWithOS(str: string, isWindows: boolean): string {

    if (!isWindows) {
        return str;
    }

    if (!str.startsWith("/")) {
        throw new Error("Parameters must begin with slash");
    }

    if (str.length <= 1) {
        throw new Error("Cannot convert string with length of 0 or 1: " + str);
    }

    const driveLetter = str.charAt(1);
    if (!isLetter(driveLetter)) {
        throw new Error("Missing drive letter: " + str + " '" + driveLetter + "'");
    }

    if (str.length === 2) {
        return driveLetter + ":\\";
    }

    const secondSlash = str.charAt(2);
    if (secondSlash !== "/") {
        throw new Error("Invalid path format: " + str);
    }

    return driveLetter + ":\\" + convertForwardSlashesToBackSlashes(str.substring(3));

}

/** Same as below, but determine behaviour based on OS. */
export function convertAbsoluteUnixStyleNormalizedPathToLocalFile(str: string): string {

    if (!isOSWindows()) {
        // For Mac/Linux, nothing to do
        return str;
    }

    return convertAbsoluteUnixStyleNormalizedPathToLocalFileWithOS(str, true);
}

function convertBackSlashesToForwardSlashes(str: string): string {
    return str.split("\\").join("/");
}

function convertForwardSlashesToBackSlashes(str: string): string {
    return str.split("/").join("\\");
}

function isOSWindows() {
    return process.platform === "win32";
}

function isLetter(currentChar: string) {
    return ("a" <= currentChar && currentChar <= "z")
        || ("A" <= currentChar && currentChar <= "Z");
}

function isUpperCase(currentChar: string) {
    return ("A" <= currentChar && currentChar <= "Z");
}

export function stripTrailingSlash(str: string): string {

    while (str.trim().endsWith("/")) {

        str = str.trim();
        str = str.substring(0, str.length - 1);

    }

    return str;

}

// Strip project parent directory from path:
// If pathToMonitor is: /home/user/codewind/project
// and watchEventPath is: /home/user/codewind/project/some-file.txt
// then this will convert watchEventPath to /some-file.txt
export function convertAbsolutePathWithUnixSeparatorsToProjectRelativePath(path: string, rootPath: string): string  {

    if (rootPath.indexOf("\\") !== -1) {
        throw new Error("Forward slashes are not supported.");
    }

    rootPath = stripTrailingSlash(rootPath);

    if (!path.startsWith(rootPath)) {
        // This shouldn't happen, and is thus severe
        log.severe("Watch event '" + path + "' does not match project path '" + rootPath + "'");
        return null;
    }

    path = path.replace(rootPath, "");

    if (path.length === 0) {
        // Ignore the empty case

        return null;
    }

    return path;

}

/**  C:\helloThere -> /c/helloThere */
export function normalizePath(pathParam: string): string {

    // Convert \ to /
    let absPath = pathParam.split("\\").join("/");

    absPath = convertFromWindowsDriveLetter(absPath);
    absPath = normalizeDriveLetter(absPath);

    return absPath;

}

/** "/moo/cow" => [ "/moo/cow", "/moo"] */
export function splitRelativeProjectPathIntoComponentPaths(path: string): string[] {
    const result: string[] = [];

    let currPath = path;
    while (true) {

        if (currPath.length === 1) {
            break;
        }

        result.push(currPath);

        const index = currPath.lastIndexOf("/");
        if (index <= 0) {
            break;
        }

        currPath = currPath.substring(0, index);
    }

    return result;
}
