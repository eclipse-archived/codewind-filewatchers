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

import path = require("path");
import { ProjectToWatch } from "./ProjectToWatch";

import * as log from "./Logger";

/**
 * This class is responsible for taking the filename/path filters for a project
 * on the watched projects list, and applying those filters against a given path
 * string (returning true if a filter should be ignored).
 */
export class PathFilter {

    private readonly _filenameExcludePatterns: RegExp[] = [];

    private readonly _pathExcludePatterns: RegExp[] = [];

    public constructor(ptw: ProjectToWatch) {

        const filenameExcludePatterns: RegExp[] = [];

        if (ptw.ignoredFilenames != null) {
            ptw.ignoredFilenames.forEach((e) => {
                if (e.indexOf("/") !== -1 || e.indexOf("\\") !== -1) {
                    log.severe("Ignored filenames may not contain path separators: " + e);
                    return;
                }

                const text = e.split("*").join(".*");

                filenameExcludePatterns.push(new RegExp(text));

            });
        }

        this._filenameExcludePatterns = filenameExcludePatterns;

        const pathExcludePatterns: RegExp[] = [];
        if (ptw.ignoredPaths != null) {
            ptw.ignoredPaths.forEach((e) => {
                if (e.indexOf("\\") !== -1) {
                    log.severe("Ignore paths may not contain Windows-style path separators: " + e);
                    return;
                }

                const text = e.split("*").join(".*");

                pathExcludePatterns.push(new RegExp(text));

            });
        }
        this._pathExcludePatterns = pathExcludePatterns;

    }

    /**
     * File parameter should be relative path from project root, rather than an
     * absolute path.
     */
    public isFilteredOutByFilename(pathParam: string): boolean {

        if (pathParam.indexOf("\\") !== -1) {
            log.severe("Path should not contain back slashes: " + pathParam);
            return false;
        }

        const strArr = pathParam.split("/");

        for (const name of strArr) {
            for (const re of this._filenameExcludePatterns) {
                if (re.test(name)) {
                    return true;
                }
            }
        }

        return false;
    }

    /**
     * File parameter should be relative path from project root, rather than an
     * absolute path.
     */
    public isFilteredOutByPath(pathParam: string): boolean {

        if (pathParam.indexOf("\\") !== -1) {
            log.severe("Parameter cannot contain Window-style file paths: " + pathParam);
            return false;
        }

        for (const re of this._pathExcludePatterns) {
            if (re.test(pathParam)) {
                return true;
            }
        }

        return false;
    }

}
