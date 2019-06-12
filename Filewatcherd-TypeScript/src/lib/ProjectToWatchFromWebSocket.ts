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

import * as models from "./Models";
import { ProjectToWatch } from "./ProjectToWatch";

export class ProjectToWatchFromWebSocket extends ProjectToWatch {

    private readonly _changeType: string;

    constructor(json: models.IWatchedProjectJson) {
        super(json, json.changeType.toLowerCase() === "delete");
        this._changeType = json.changeType;
    }

    public get changeType(): string {
        return this._changeType;
    }
}
