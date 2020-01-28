/*******************************************************************************
* Copyright (c) 2020 IBM Corporation and others.
* All rights reserved. This program and the accompanying materials
* are made available under the terms of the Eclipse Public License v2.0
* which accompanies this distribution, and is available at
* http://www.eclipse.org/legal/epl-v20.html
*
* Contribu	rs:
*     IBM Corporation - initial API and implementation
*******************************************************************************/

package main

import (
	"codewind/utils"
	"os"
	"strconv"
	"time"
)

// IndividualFileWatchService is used to watch a small number of individual files, for example,
// linked files defined in the 'refPaths' field of a watched project. For a
// large number of files to watch, the watch service should be used instead.
//
// Files watched by this class do not need to exist.
//
// A single instance of this class will exist per filewatcher (eg it is not per
// project).
//
// This class was introduced as part of 'Project sync support for reference to
// files outside of project folder ' (codewind/1399).
type IndividualFileWatchService struct {
	cmdChannel  chan indivFileWatchServiceCmd
	projectList *ProjectList
}

type indivFileWatchServiceCmdType int

const (
	iwsSetFilesToWatchCmd = iota + 1
	iwsTimerTickCmd
	iwsWatchServiceDispose
)

type indivFileWatchServiceCmd struct {
	cmdType      indivFileWatchServiceCmdType
	projectID    string
	pathsFromPtw []string
}

// NewIndividualFileWatchService creates an instance of this service. Only one should exist per process.
func NewIndividualFileWatchService(projectList *ProjectList) *IndividualFileWatchService {

	result := &IndividualFileWatchService{
		cmdChannel:  make(chan indivFileWatchServiceCmd),
		projectList: projectList,
	}

	go result.commandReceiver()

	return result
}

// SetFilesToWatch is used to inform the command receiver of the most recent set of files that it should be watching, for each project.
func (ifws *IndividualFileWatchService) SetFilesToWatch(projectID string, pathsFromPtw []string) {
	ifws.cmdChannel <- indivFileWatchServiceCmd{cmdType: iwsSetFilesToWatchCmd, projectID: projectID, pathsFromPtw: pathsFromPtw}
}

func (ifws *IndividualFileWatchService) commandReceiver() {
	filesToWatchMap := make(map[string] /*project id*/ (map[string] /*absolute path*/ *pollEntry /*linked files*/))

	// Start the channel timer
	ifws.timerTick(filesToWatchMap, ifws.projectList)

	disposed := false

	for {
		select {
		case cmd := <-ifws.cmdChannel:

			if disposed {
				// We intentionally don't exit on dispose: doing so will blocking channel senders. The channel will be GC-ed
				// by the runtime when there are no more consumers.
				continue
			}

			if cmd.cmdType == iwsSetFilesToWatchCmd {
				handleSetFilesToWatch(cmd.projectID, cmd.pathsFromPtw, filesToWatchMap)

			} else if cmd.cmdType == iwsTimerTickCmd {
				ifws.timerTick(filesToWatchMap, ifws.projectList)

			} else if cmd.cmdType == iwsWatchServiceDispose {
				disposed = true
				for k := range filesToWatchMap {
					delete(filesToWatchMap, k)
				}
			}
		}
	}
}

// Each X seconds, in this method we scan the list of files to watch and report changes.
func (ifws *IndividualFileWatchService) timerTick(filesToWatchMap map[string](map[string]*pollEntry), projectList *ProjectList) {

	// Maintain a list of file changes, per project
	fileChangesDetected := make(map[string] /*project id*/ []ChangedFileEntry)

	for projectID, filesToWatch := range filesToWatchMap {

		// For each of the watches files for the specified projectID...
		for _, fileToWatch := range filesToWatch {

			fileModifiedTime := int64(0)
			fileExists := true
			file, err := os.Stat(fileToWatch.absolutePath)
			if os.IsNotExist(err) {
				fileExists = false
			}

			newStatus := pollEntryStatus(pollEntryStatusExists)
			if !fileExists {
				newStatus = pollEntryStatusDoesNotExist
			} else {
				fileModifiedTime = file.ModTime().UnixNano() / 1000000
			}

			if fileToWatch.lastObservedStatus != pollEntryStatusRecentlyAdded {

				if fileToWatch.lastObservedStatus != newStatus {

					eventType := "CREATE"

					if fileExists {
						// ADDED: Last time we saw this file it did not exist, but now it does.
						utils.LogInfo("Watched file now exists: " + fileToWatch.absolutePath)
						eventType = "CREATE"
					} else {
						// DELETED: Last time we saw this file it did exist, but it no longer does.
						utils.LogInfo("Watched file has been deleted: " + fileToWatch.absolutePath)
						eventType = "DELETE"
					}

					changedFiles, exists := fileChangesDetected[projectID]
					if !exists {
						changedFiles = []ChangedFileEntry{}
					}

					changedFileEntry, err := NewChangedFileEntry(fileToWatch.absolutePath, eventType, time.Now().UnixNano()/1000000, false)
					if err != nil {
						utils.LogSevereErr("Unable to create changed file entry", err)
						continue
					}

					changedFiles = append(changedFiles, *changedFileEntry)

					fileChangesDetected[projectID] = changedFiles

				}
				if fileModifiedTime > 0 && fileToWatch.lastModifiedTime > 0 && fileModifiedTime != fileToWatch.lastModifiedTime {

					// CHANGED: Last time we same this file it had a different modified time.
					utils.LogInfo("Watched file change detected: " + fileToWatch.absolutePath + " " + strconv.FormatInt(fileModifiedTime, 10) + " " + strconv.FormatInt(fileToWatch.lastModifiedTime, 10))

					changedFiles, exists := fileChangesDetected[projectID]
					if !exists {
						changedFiles = []ChangedFileEntry{}
					}

					changedFileEntry, err := NewChangedFileEntry(fileToWatch.absolutePath, "MODIFY", time.Now().UnixNano()/1000000, false)
					if err != nil {
						utils.LogSevereErr("Unable to create changed file entry", err)
						continue
					}
					changedFiles = append(changedFiles, *changedFileEntry)
					fileChangesDetected[projectID] = changedFiles

				}
			} // end of if recentlyModified block

			fileToWatch.lastObservedStatus = newStatus
			fileToWatch.lastModifiedTime = fileModifiedTime
		} // end filesToWatch iteration
	} // end filesToWatchMap iteration

	// For each of the changes we found, communicate them to the EventBatchUtil, via the project list.
	for projectID, paths := range fileChangesDetected {

		if len(paths) == 0 {
			continue
		}

		projectList.ReceiveIndividualChangesFileList(projectID, paths)

	}

	// Trigger a new file checked timer tick X seconds after the previous one finishes.
	go func() {
		time.Sleep(time.Second * 2)
		ifws.cmdChannel <- indivFileWatchServiceCmd{cmdType: iwsTimerTickCmd, projectID: "", pathsFromPtw: []string{}}

	}()
}

// In this method we synchronize the paths that FW is telling us we should be watching, with what we are currently watching.
func handleSetFilesToWatch(projectID string, pathsFromPtw []string, filesToWatchMap map[string](map[string]*pollEntry)) {

	// Convert pathsFromPtw param to a platform-specific format (if needed), and filter out dirs
	paths := []string{}
	for _, pathFromPtw := range pathsFromPtw {
		newPath, err := utils.ConvertAbsoluteUnixStyleNormalizedPathToLocalFile(pathFromPtw)
		if err != nil {
			utils.LogSevereErr("Unable to convert path: "+pathFromPtw, err)
			continue
		}

		// Filter out and report directories
		if info, err := os.Stat(newPath); err == nil && info.IsDir() {
			utils.LogError("Project '" + projectID + "' was asked to watch a directory, which is not supported: " + newPath)
			continue
		}

		paths = append(paths, newPath)
	}

	if len(paths) == 0 {
		return
	}

	// mapUpdated := false

	currProjectState, exists := filesToWatchMap[projectID]
	if !exists {
		// This is a new project we haven't seen.
		newFiles := make(map[string]*pollEntry)

		for _, path := range paths {
			pollEntry := &pollEntry{lastObservedStatus: pollEntryStatusRecentlyAdded, absolutePath: path, lastModifiedTime: 0}
			newFiles[pollEntry.absolutePath] = pollEntry
			utils.LogInfo("Files to watch - recently added for new project: " + path)
		}
		// mapUpdated = true

		filesToWatchMap[projectID] = newFiles
	} else {
		// This is an existing project with at least one file we are currently monitoring.

		for _, path := range paths {

			_, exists := currProjectState[path]
			if !exists {
				utils.LogInfo("Files to watch - recently added for existing project: " + path)
				currProjectState[path] = &pollEntry{lastObservedStatus: pollEntryStatusRecentlyAdded, absolutePath: path, lastModifiedTime: 0}
				// mapUpdated = true
			} else {
				// Ignore: the path is in both maps -- no change.
			}
		}

		pathsInParam := make(map[string] /*absolute path*/ bool /*unused*/)
		for _, path := range paths {
			pathsInParam[path] = true
		}

		keysToRemove := []string{}

		// Look for values that are in curr project state, but not in the parameter list. These are
		// files that we WERE watching, but are no longer.
		for pathInCurrentState := range currProjectState {

			_, exists := pathsInParam[pathInCurrentState]
			if !exists {
				keysToRemove = append(keysToRemove, pathInCurrentState)
				utils.LogInfo("Files to watch - removing from watch list: " + pathInCurrentState)
				// mapUpdated = true
			}
		}

		for _, keyToRemove := range keysToRemove {
			delete(currProjectState, keyToRemove)
		}

		// If we're not watching anything anymore, entirely remove the project from the state list.
		if len(currProjectState) == 0 {
			delete(filesToWatchMap, projectID)
		}
	} // end existing project else

} // end function

type pollEntryStatus int

const (
	pollEntryStatusRecentlyAdded = iota + 1
	pollEntryStatusExists
	pollEntryStatusDoesNotExist
)

type pollEntry struct {
	lastObservedStatus pollEntryStatus
	absolutePath       string
	lastModifiedTime   int64
}
