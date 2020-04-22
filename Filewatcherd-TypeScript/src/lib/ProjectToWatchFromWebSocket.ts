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

import * as models from "./Models";
import { ProjectToWatch } from "./ProjectToWatch";

/**
 * The watcher WebSocket will inform of us of project watch changes just like
 * the GET watchlist API, except that the WebSocket JSON has an additional field
 * 'changeType' which specifies how the watched project changed: for example, if
 * a project is deleted or created, this field will indicate as such.
 *
 * Absent the above field, this object has the same structure as its parent.
 */
export class ProjectToWatchFromWebSocket extends ProjectToWatch {

    public get changeType(): string {
        return this._changeType;
    }

    public static cloneWebSocketWithNewProjectCreationTime(old: ProjectToWatchFromWebSocket,
                                                           projectCreationTimeInAbsoluteMsecsParam: number)
        : ProjectToWatchFromWebSocket {

        const result = new ProjectToWatchFromWebSocket(old._changeType);

        ProjectToWatch.copyWithNewProjectCreationTime(result, old, projectCreationTimeInAbsoluteMsecsParam);
        return result;
    }

    public static create(json: models.IWatchedProjectJson): ProjectToWatchFromWebSocket {
        const result = new ProjectToWatchFromWebSocket(json.changeType);
        ProjectToWatch.innerCreateFromJson(result, json, json.changeType.toLowerCase() === "delete");

        return result;
    }

    private _changeType: string;

    protected constructor(changeType : string) {
        super();
        this._changeType = changeType;
    }
}
