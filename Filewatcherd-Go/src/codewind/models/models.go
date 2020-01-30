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

package models

// ProjectToWatch ...
type ProjectToWatch struct {
	IgnoredFilenames    []string       `json:"ignoredFilenames"`
	IgnoredPaths        []string       `json:"ignoredPaths"`
	PathToMonitor       string         `json:"pathToMonitor"`
	ProjectID           string         `json:"projectID"`
	ChangeType          string         `json:"changeType"`
	ProjectWatchStateID string         `json:"projectWatchStateId"`
	Type                string         `json:"type"`
	ProjectCreationTime int64          `json:"projectCreationTime"`
	RefPaths            []RefPathEntry `json:"refPaths"`
}

// RefPathEntry ...
type RefPathEntry struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ConvertRefPathsToFromStrings is a simple utility method that converts RefPaths to an array containing only the From field of each entry
func ConvertRefPathsToFromStrings(ptw *ProjectToWatch) []string {
	indivFilesToWatch := []string{}
	for _, refPath := range ptw.RefPaths {
		indivFilesToWatch = append(indivFilesToWatch, refPath.From)
	}
	return indivFilesToWatch
}

// Clone performs a deep copy of a ProjectToWatch
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

	var newRefPaths []RefPathEntry
	if entry.RefPaths != nil {
		newRefPaths = []RefPathEntry{}
		for _, val := range entry.RefPaths {
			newRefPaths = append(newRefPaths, RefPathEntry{From: val.From, To: val.To})
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
		entry.ProjectCreationTime,
		newRefPaths,
	}
}

// WatchlistEntries ...
type WatchlistEntries []ProjectToWatch

// WatchlistEntryList ...
type WatchlistEntryList struct {
	Projects WatchlistEntries `json:"projects"`
}

// WatchEventEntry ...
type WatchEventEntry struct {
	EventType string
	Path      string
	IsDir     bool
}

// WatchChangeJson ...
type WatchChangeJson struct {
	Type     string           `json:"type"`
	Projects WatchlistEntries `json:"projects"`
}
