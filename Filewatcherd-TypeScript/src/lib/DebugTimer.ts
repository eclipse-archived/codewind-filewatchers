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

import { FileWatcher } from "./FileWatcher";
import * as log from "./Logger";

/**
 * Every X minutes, this timer will run and output the internal state of
 * each of the internal components of the filewatcher. This should run
 * infrequently, as it can be fairly verbose (eg every 30 minutes).
 *
 * The goal of this timer is to identify bugs/performance issues that might only
 * be detectable when the filewatcher is running for long periods of time (memory
 * leaks, resources we aren't closing, etc).
 */
export class DebugTimer {

    private readonly _parent: FileWatcher;

    constructor(parent: FileWatcher) {
        this._parent = parent;
    }

    private tick() {

        let renew = true;

        try {

            const debugStr = this._parent.generateDebugString();
            if (!debugStr || debugStr.trim().length === 0) {
                // Filewatcher is disposed.
                renew = false;
                return;
            }

            for (const str of debugStr.split("\n"))  {
                log.info("[status] " + str);
            }

        } finally {
            if (renew) {
                this.schedule();
            }
        }

    }

    public schedule() {
        const debugTimer = this;
        setTimeout(() => {
            debugTimer.tick();
        }, 30 * 60 * 1000);

    }
}
