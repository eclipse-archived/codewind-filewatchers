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

/** This is not currently used, but I reserve the right to clone all the things at a later date. */
func (entry *ProjectToWatch) Clone() *ProjectToWatch {

	var newIgnoredFilenames []string

	if entry.IgnoredFilenames != nil {
		newIgnoredFilenames = make([]string, 0)
		for _, val := range entry.IgnoredFilenames {
			newIgnoredFilenames = append(newIgnoredFilenames, val)
		}
	}

	var newIgnoredPaths []string

	if entry.IgnoredPaths != nil {
		newIgnoredPaths = make([]string, 0)
		for _, val := range entry.IgnoredPaths {
			newIgnoredPaths = append(newIgnoredPaths, val)
		}
	}

	return &ProjectToWatch{
		newIgnoredFilenames,
		newIgnoredPaths,
		entry.PathToMonitor,
		entry.ProjectID,
		entry.ChangeType,
		entry.ProjectWatchStateID,
		entry.Type,
	}
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
