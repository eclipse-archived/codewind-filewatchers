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
import { FileWatcher } from "./FileWatcher";
import { ProjectToWatch } from "./ProjectToWatch";

export interface IWatchService {

    addPath(fileToMonitor: string, ptw: ProjectToWatch): void;

    removePath(fileToMonitor: string, oldProjectToWatch: ProjectToWatch): void;

    setParent(parent: FileWatcher): void;

    dispose(): void;

    generateDebugState(): string;

}
