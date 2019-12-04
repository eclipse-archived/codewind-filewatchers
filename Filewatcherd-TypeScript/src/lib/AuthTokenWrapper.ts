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

import { FWAuthToken } from "./FWAuthToken";
import { IAuthTokenProvider } from "./IAuthTokenProvider";

import * as log from "./Logger";

/**
 * AuthTokenWrapper is the conduit through the internal filewatcher codebase
 * requests secure authentication tokens from the IDE. In cases where the
 * authTokenProvider is null (eg is secure auth is not required), the methods of
 * this class are no-ops.
 *
 * This class was created as part of issue codewind/1309.
 */
export class AuthTokenWrapper {

    public static readonly KEEP_LAST_X_STALE_KEYS = 10;

    /**
     * Contains an ordered (descending by creation time) list of invalids keys, with
     * at most KEEP_LAST_X_STALE_KEYS keys.
     */
    private _recentInvalidKeysQueue: FWAuthToken[];

    /**
     * Contains invalid keys; used as a fast path to determine if a given key is
     * already invalid. The should be at most KEEP_LAST_X_STALE_KEYS here.
     */
    private _invalidKeysSet: Set<string /* token */>;

    private readonly _authTokenProvider: IAuthTokenProvider;

    constructor(authTokenProvider: IAuthTokenProvider) {
        this._authTokenProvider = authTokenProvider;
        this._invalidKeysSet = new Set();
    }
    public getLatestToken(): FWAuthToken {
        if (!this._authTokenProvider) { return null; }

        const token = this._authTokenProvider.getLatestAuthToken();
        if (!token) { return null; }

        log.info("IDE returned a new security token to filewatcher: " + this.digest(token));

        return token;
    }

    /** Inform the IDE when a token is rejected. */
    public informBadToken(token: FWAuthToken): void {
        if (!this._authTokenProvider || !token || !token.accessToken) {
            return;
        }

        // We've already reported this key as invalid, so just return it
        if (this._invalidKeysSet.has(token.accessToken)) {
            log.info("Filewatcher informed us of a bad token, but we've already reported it to the IDE: "
                + this.digest(token));
            return;
        }

        // We have a new token that we have not previously reported as invalid.

        this._invalidKeysSet.add(token.accessToken);
        this._recentInvalidKeysQueue.push(token); // append to end

        while (this._recentInvalidKeysQueue.length > AuthTokenWrapper.KEEP_LAST_X_STALE_KEYS) {

            const keyToRemove = this._recentInvalidKeysQueue.shift(); // remove from front
            this._invalidKeysSet.delete(keyToRemove.accessToken);
        }

    }

    /**
     * Return a representation of the token that is at most 32 characters long, so
     * as not to overwhelm the log file.
     */
    private digest(token: FWAuthToken): string {
        if (!token) { return null; }

        const key = token.accessToken;
        if (!key) { return null; }

        return key.substring(0, Math.min(key.length, 32));

    }

}
