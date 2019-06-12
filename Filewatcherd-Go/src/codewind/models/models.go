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

package models

type ProjectToWatch struct {
	IgnoredFilenames           []string `json:"ignoredFilenames"`
	IgnoredPaths               []string `json:"ignoredPaths"`
	PathToMonitor              string   `json:"pathToMonitor"`
	ProjectID                  string   `json:"projectID"`
	ChangeType                 string   `json:"changeType"`
	ProjectWatchStateID        string   `json:"projectWatchStateId"`
	Type                       string   `json:"type"`
}

type WatchlistEntries []ProjectToWatch

type WatchlistEntryList struct {
	Projects WatchlistEntries `json:"projects"`
}

type WatchEventEntry struct {
	EventType string
	Path      string
	IsDir     bool
}

type WatchChangeJson struct {
	Type     string           `json:"type"`
	Projects WatchlistEntries `json:"projects"`
}
