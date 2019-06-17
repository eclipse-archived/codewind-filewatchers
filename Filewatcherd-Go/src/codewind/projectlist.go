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

package main

import (
	"codewind/models"
	"codewind/utils"
	"time"
)

type ProjectList struct {
	projectOperationChannel chan *projectListChannelMessage
}


type receiveNewWatchEntriesMessage struct {
	watchEventEntry *models.WatchEventEntry
	project *models.ProjectToWatch
}

func NewProjectList(postOutputQueue *HttpPostOutputQueue) *ProjectList {

	result := &ProjectList{}
	result.projectOperationChannel = make(chan *projectListChannelMessage)
	go result.channelListener(postOutputQueue)

	return result
}

type projectListMessageType int

const (
	SetWatchServiceMsg = iota + 1
	UpdateProjectListFromWebSocketMsg
	UpdateProjectListFromGetRequestMsg
	ReceiveNewWatchEventEntriesMsg
)

type projectListChannelMessage struct {
	msgType                                projectListMessageType
	setWatchServiceMessage                 *WatchService
	updateProjectListFromWebSocketMessage  *models.WatchChangeJson
	updateProjectListFromGetRequestMessage *models.WatchlistEntries
	receiveNewWatchEventEntriesMessage     *receiveNewWatchEntriesMessage
}

func (projectList *ProjectList) SetWatchService(watchService *WatchService) {

	projectList.projectOperationChannel <- &projectListChannelMessage{
		SetWatchServiceMsg,
		watchService,
		nil,
		nil,
		nil,
	}

}

func (projectList *ProjectList) UpdateProjectListFromWebSocket(watchChange *models.WatchChangeJson) {
	projectList.projectOperationChannel <- &projectListChannelMessage{
		UpdateProjectListFromWebSocketMsg,
		nil,
		watchChange,
		nil,
		nil,
	}
}

func (projectList *ProjectList) UpdateProjectListFromGetRequest(entries *models.WatchlistEntries) {
	projectList.projectOperationChannel <- &projectListChannelMessage{
		UpdateProjectListFromGetRequestMsg,
		nil,
		nil,
		entries,
		nil,
	}
}

func (projectList *ProjectList) ReceiveNewWatchEventEntries(entry *models.WatchEventEntry, project *models.ProjectToWatch) {

	rnwem := &receiveNewWatchEntriesMessage  {
		entry,
		project,
	}

	projectList.projectOperationChannel <- &projectListChannelMessage{
		ReceiveNewWatchEventEntriesMsg,
		nil,
		nil,
		nil,
		rnwem,
	}
}

func (projectList *ProjectList) channelListener(postOutputQueue *HttpPostOutputQueue) {

	/** projectId -> most recent watch list for a project */
	var projectsMap map[string]*projectObject
	projectsMap = make(map[string]*projectObject)

	var watchService *WatchService

	for {

		select {
		case projectOperationMessage := <-projectList.projectOperationChannel:
			if projectOperationMessage.msgType == SetWatchServiceMsg {
				watchService = projectOperationMessage.setWatchServiceMessage

			} else if projectOperationMessage.msgType == UpdateProjectListFromWebSocketMsg {
				projectList.handleUpdateProjectListFromWebSocket(projectOperationMessage.updateProjectListFromWebSocketMessage, projectsMap, watchService, postOutputQueue)

			} else if projectOperationMessage.msgType == UpdateProjectListFromGetRequestMsg {
				projectList.handleUpdateProjectListFromGetRequest(projectOperationMessage.updateProjectListFromGetRequestMessage, projectsMap, watchService, postOutputQueue)

			} else if projectOperationMessage.msgType == ReceiveNewWatchEventEntriesMsg {
				msg := projectOperationMessage.receiveNewWatchEventEntriesMessage
				handleReceiveNewWatchEventEntries(msg.project, msg.watchEventEntry, projectsMap)
			}
		}

	}
}

func (projectList *ProjectList) handleUpdateProjectListFromGetRequest(entries *models.WatchlistEntries, projectsMap map[string]*projectObject, watchService *WatchService, postOutputQueue *HttpPostOutputQueue) {

	// Delete projects that are not in the entries list
	// - We do delete first, so as not to interfere with the 'create projects' step below it, 
	//   that may share the same path.

	/** project id -> true*/
	var projectIDInHTTPResult map[string]bool

	removedProjects := make([]*projectObject, 0)

	projectIDInHTTPResult = make(map[string]bool)

	for _, project := range *entries {
		
		_, exists := projectIDInHTTPResult[project.ProjectID]
		if exists {
			utils.LogSevere("Multiple projects in the project list share the same project ID: " + project.ProjectID)
		}

		projectIDInHTTPResult[project.ProjectID] = true
	}

	for key, val := range projectsMap {
		// For each of the projects in the local state map, if they aren't found
		// in the HTTP GET result, then they have been removed.
		_, exists := projectIDInHTTPResult[key]

		if !exists {
			removedProjects = append(removedProjects, val)
		}

	}

	for _, removedProject := range removedProjects {

		utils.LogInfo("Removing project from watch list from GET: " + removedProject.project.ProjectID + " " + removedProject.project.PathToMonitor)
		delete(projectsMap, removedProject.project.ProjectID)
	}

	for _, removedProject := range removedProjects {
		fileToMonitor, err := utils.ConvertAbsoluteUnixStyleNormalizedPathToLocalFile(removedProject.project.PathToMonitor)
		if err != nil {
			utils.LogSevereErr("Unable to convert path after project remove", err)
			continue
		}
		utils.LogDebug("Calling watch service removePath with file: " + fileToMonitor)

		watchService.RemoveRootPath(fileToMonitor, removedProject.project)
	}

	// Next, create new projects, or updating existing ones
	for _, project := range *entries {
		processProject(&project, projectsMap, postOutputQueue, watchService)
	}

}

func (projectList *ProjectList) handleUpdateProjectListFromWebSocket(webSocketUpdates *models.WatchChangeJson, projectsMap map[string]*projectObject, watchService *WatchService, postOutputQueue *HttpPostOutputQueue) {

	utils.LogInfo("Processing a received file watch state from WebSocket")

	for _, projectFromWS := range webSocketUpdates.Projects {

		if projectFromWS.ChangeType == "delete" {
			currProjWatchState, exists := projectsMap[projectFromWS.ProjectID]
			if exists {
				utils.LogInfo("Removing project from watch list: " + currProjWatchState.project.ProjectID + " " + currProjWatchState.project.PathToMonitor)

				delete(projectsMap, projectFromWS.ProjectID)
			} else {
				utils.LogError("Unable to find deleted project from WebSocket in project map: " + projectFromWS.ProjectID)
			}
			pathToRemove, err := utils.ConvertAbsoluteUnixStyleNormalizedPathToLocalFile(currProjWatchState.project.PathToMonitor)
			if err != nil {
				utils.LogSevere("Unable to convert path to absolute unix style path" + pathToRemove)
			} else {
				utils.LogDebug("Calling watch service removePath with file: " + pathToRemove)
				if watchService != nil {
					watchService.RemoveRootPath(pathToRemove, &projectFromWS)
				} else {
					utils.LogSevere("Watch service is not set in project list and a RemoveRootPath was missed: " + pathToRemove)
				}
			}

		} else {
			processProject(&projectFromWS, projectsMap, postOutputQueue, watchService)
		}
	}

}

func processProject(projectToProcess *models.ProjectToWatch, projectsMap map[string]*projectObject, postOutputQueue *HttpPostOutputQueue, watchService *WatchService) {

	currProjWatchState, exists := projectsMap[projectToProcess.ProjectID]
	if exists {
		// If we have previously monitored this project...

		if currProjWatchState.project.PathToMonitor == projectToProcess.PathToMonitor {

			oldProjectToWatch := currProjWatchState.project

			fileToMonitor, err := utils.ConvertAbsoluteUnixStyleNormalizedPathToLocalFile(projectToProcess.PathToMonitor)
			if err != nil {
				utils.LogSevereErr("Unable to convert from absolute unix style normalized path: "+projectToProcess.PathToMonitor, err)
				return
			}

			// If the watch has changed, then remove the path and update the PTW
			if oldProjectToWatch.ProjectWatchStateID != projectToProcess.ProjectWatchStateID {

				utils.LogInfo("The project watch state has changed: " + oldProjectToWatch.ProjectWatchStateID + " " + projectToProcess.ProjectWatchStateID + " for project " + projectToProcess.ProjectID)

				// Update the map with the value from the web socket
				projectToProcess.ChangeType = ""
				currProjWatchState.project = projectToProcess

				// We remove, then add, the watcher here, because the filters may have changed.

				// Remove the old path
				watchService.RemoveRootPath(fileToMonitor, projectToProcess)
				utils.LogInfo("From update, removed project with path '" + projectToProcess.PathToMonitor + "' from watch list, with watch directory: '" + fileToMonitor + "'")

				// Added the new path and PTW
				watchService.AddRootPath(fileToMonitor, projectToProcess)
				utils.LogInfo("From update, added new project with path '" + projectToProcess.PathToMonitor + "' to watch list, with watch directory: '" + fileToMonitor + "'")
			} else {
				utils.LogInfo("The project watch state has not changed for project " + projectToProcess.ProjectID)
			}

		} else {
			utils.LogSevere("The path to monitor of a project cannot be changed once it set, for a particular project id")
		}

	} else {
		// This is the first time we are hearing about this project

		currProjWatchState = newProjectObject(projectToProcess, postOutputQueue)
		projectsMap[projectToProcess.ProjectID] = currProjWatchState

		// For Windows, the server will give us path in the form of '/c/Users/Administrator',
		// which we need to convert to 'c:\Users\Administrator', below.
		fileToMonitor, err := utils.ConvertAbsoluteUnixStyleNormalizedPathToLocalFile(currProjWatchState.project.PathToMonitor)
		if err != nil {
			utils.LogSevereErr("Unable to convert from absolute unix style normalized path: "+currProjWatchState.project.PathToMonitor, err)
		} else {
			if watchService != nil {
				watchService.AddRootPath(fileToMonitor, projectToProcess)
				utils.LogDebug("Added new project with path '" + projectToProcess.PathToMonitor + "' to watch list, with watch directory: '" + fileToMonitor + "'")
			} else {
				utils.LogSevere("Watch service is not set in project list and an AddRootPath was missed: " + fileToMonitor)
			}
		}
	}

}


func handleReceiveNewWatchEventEntries(projectMatch *models.ProjectToWatch, entry *models.WatchEventEntry, projectsMap map[string]*projectObject) {

	utils.LogDebug("Received new watch entry: " + entry.EventType + " " + entry.Path+" " + projectMatch.ProjectID)

	filter, err := utils.NewPathFilter(projectMatch)
	if err != nil {
		utils.LogSevere("Could not create filter for " + projectMatch.ProjectID)
		return
	}

	path := utils.ConvertAbsolutePathWithUnixSeparatorsToProjectRelativePath(entry.Path, projectMatch.PathToMonitor)

	if path == nil || len(*path) == 0 {
		return
	}

	if projectMatch.IgnoredPaths != nil && filter.IsFilteredOutByPath(*path) {
		utils.LogDebug("Filtered out '" + *path + "' due to path filter")
		return
	}

	if projectMatch.IgnoredFilenames != nil && filter.IsFilteredOutByFilename(*path) {
		utils.LogDebug("Filtered out '" + *path + "' due to filename filter")
		return
	}

	val, exists := projectsMap[projectMatch.ProjectID]
	if exists {
		entry, err := NewChangedFileEntry(*path, entry.EventType, time.Now().UnixNano()/1000000, entry.IsDir)
		if err != nil {
			utils.LogSevereErr("Error in creating new changed file entry", err)
			return
		}

		changedFileEntries := []ChangedFileEntry{*entry}

		val.eventBatchUtil.AddChangedFiles(changedFileEntries)
	} else {
		utils.LogSevere("Could not locate event processing for project id " + projectMatch.ProjectID)
		return
	}

}

type projectObject struct {
	project        *models.ProjectToWatch
	eventBatchUtil *FileChangeEventBatchUtil
}

func newProjectObject(project *models.ProjectToWatch, postOutputQueue *HttpPostOutputQueue) *projectObject {
	return &projectObject{
		project,
		NewFileChangeEventBatchUtil(project.ProjectID, postOutputQueue),
	}
}
