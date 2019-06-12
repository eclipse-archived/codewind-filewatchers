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

export class ExponentialBackoffUtil {

    public static getDefaultBackoffUtil(maxFailureDelay: number): ExponentialBackoffUtil {
        return new ExponentialBackoffUtil(500, maxFailureDelay, 1.5);
    }

    private readonly _minFailureDelay: number;

    private _failureDelay: number;

    private readonly _maxFailureDelay: number;

    private readonly _backoffExponent: number;

    constructor(minFailureDelay: number, maxFailureDelay: number, backoffExponent: number) {
        this._minFailureDelay = minFailureDelay;
        this._maxFailureDelay = maxFailureDelay;
        this._failureDelay = minFailureDelay;
        this._backoffExponent = backoffExponent;
    }

    public async sleepAsync() {
        await new Promise( (resolve) => setTimeout(resolve, this._failureDelay) );
    }

    public failIncrease() {
        this._failureDelay *= this._backoffExponent;
        if (this._failureDelay > this._maxFailureDelay) {
            this._failureDelay = this._maxFailureDelay;
        }
    }
    public successReset() {
        this._failureDelay = this._minFailureDelay;
    }

}
