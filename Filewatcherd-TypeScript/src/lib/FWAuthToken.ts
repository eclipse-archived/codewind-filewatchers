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

/**
 * Immutable authorization token for use in HTTP(S) requests and WebSocket
 * connections
 */
export class FWAuthToken {

    private readonly _accessToken: string;

    private readonly _tokenType: string;

    constructor(accessToken: string, tokenType: string) {
        if (!accessToken || !tokenType) {
            throw new Error("Invalid parameters to FWAuthToken: " + accessToken + " " + tokenType);
        }

        this._accessToken = accessToken;
        this._tokenType = tokenType;
    }

    public get accessToken(): string {
        return this._accessToken;
    }

    public get tokenType(): string {
        return this._tokenType;
    }

}
