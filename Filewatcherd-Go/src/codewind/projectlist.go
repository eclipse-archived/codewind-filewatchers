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
	"strings"
	"time"
)

/**
 * ProjectList is the API entrypoint for other code in this application to perform operations against monitored projects:
 * - Update project list from a GET response
 * - Update project list from a WebSocket response
 * - Process a file update and pass it to batch utility
 *
 * Behind the scenes, the ProjectList API calls are translated into channel messages and placed on the projectOperationChannel.
 * This allows us to provide thread safety to the internal project list data, as that data will only ever be accessed
 * by a single goroutine.
 */
type ProjectList struct {
	projectOperationChannel chan *projectListChannelMessage
	pathToInstaller         string // nullable
}

type receiveNewWatchEntriesMessage struct {
	watchEventEntry *models.WatchEventEntry
	project         *models.ProjectToWatch
}

func NewProjectList(postOutputQueue *HttpPostOutputQueue, pathToInstallerParam string) *ProjectList {

	result := &ProjectList{}
	result.projectOperationChannel = make(chan *projectListChannelMessage)
	result.pathToInstaller = pathToInstallerParam
	go result.channelListener(postOutputQueue)

	return result
}

type projectListMessageType int

const (
	setWatchServiceMsg = iota + 1
	updateProjectListFromWebSocketMsg
	updateProjectListFromGetRequestMsg
	receiveNewWatchEventEntriesMsg
	requestDebugMsg
	cliFileChangeUpdate
)

type projectListChannelMessage struct {
	msgType                                projectListMessageType
	setWatchServiceMessage                 *WatchService
	updateProjectListFromWebSocketMessage  *models.WatchChangeJson
	updateProjectListFromGetRequestMessage *models.WatchlistEntries
	receiveNewWatchEventEntriesMessage     *receiveNewWatchEntriesMessage
	requestDebugMessage                    chan string
	cliFileChangeUpdateMessage             string // project id
}

func (projectList *ProjectList) SetWatchService(watchService *WatchService) {

	projectList.projectOperationChannel <- &projectListChannelMessage{
		msgType:                setWatchServiceMsg,
		setWatchServiceMessage: watchService,
	}

}

func (projectList *ProjectList) UpdateProjectListFromWebSocket(watchChange *models.WatchChangeJson) {
	projectList.projectOperationChannel <- &projectListChannelMessage{
		msgType:                               updateProjectListFromWebSocketMsg,
		updateProjectListFromWebSocketMessage: watchChange,
	}
}

func (projectList *ProjectList) UpdateProjectListFromGetRequest(entries *models.WatchlistEntries) {
	projectList.projectOperationChannel <- &projectListChannelMessage{
		msgType:                                updateProjectListFromGetRequestMsg,
		updateProjectListFromGetRequestMessage: entries,
	}
}

func (projectList *ProjectList) RequestDebugMessage() chan string {
	result := make(chan string)
	projectList.projectOperationChannel <- &projectListChannelMessage{
		msgType:             requestDebugMsg,
		requestDebugMessage: result,
	}
	return result

}

func (projectList *ProjectList) ReceiveNewWatchEventEntries(entry *models.WatchEventEntry, project *models.ProjectToWatch) {

	rnwem := &receiveNewWatchEntriesMessage{
		entry,
		project,
	}

	projectList.projectOperationChannel <- &projectListChannelMessage{
		msgType:                            receiveNewWatchEventEntriesMsg,
		receiveNewWatchEventEntriesMessage: rnwem,
	}
}

func (projectList *ProjectList) CLIFileChangeUpdate(projectId string) {

	projectList.projectOperationChannel <- &projectListChannelMessage{
		msgType:                    cliFileChangeUpdate,
		cliFileChangeUpdateMessage: projectId,
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
			if projectOperationMessage.msgType == setWatchServiceMsg {
				watchService = projectOperationMessage.setWatchServiceMessage

			} else if projectOperationMessage.msgType == updateProjectListFromWebSocketMsg {
				projectList.handleUpdateProjectListFromWebSocket(projectOperationMessage.updateProjectListFromWebSocketMessage, projectsMap, watchService, postOutputQueue)

			} else if projectOperationMessage.msgType == updateProjectListFromGetRequestMsg {
				projectList.handleUpdateProjectListFromGetRequest(projectOperationMessage.updateProjectListFromGetRequestMessage, projectsMap, watchService, postOutputQueue)

			} else if projectOperationMessage.msgType == receiveNewWatchEventEntriesMsg {
				msg := projectOperationMessage.receiveNewWatchEventEntriesMessage
				handleReceiveNewWatchEventEntries(msg.project, msg.watchEventEntry, projectsMap)

			} else if projectOperationMessage.msgType == requestDebugMsg {
				responseChan := projectOperationMessage.requestDebugMessage
				responseChan <- projectList.handleRequestDebugMsg(projectsMap)
			} else if projectOperationMessage.msgType == cliFileChangeUpdate {
				projectList.handleCliFileChangeUpdate(projectOperationMessage.cliFileChangeUpdateMessage, projectsMap)
			}
		}

	}
}

/** Generate an overview of the state of the project list, including the projects being watched. */
func (projectList *ProjectList) handleCliFileChangeUpdate(projectID string, projectsMap map[string]*projectObject) {

	value, exists := projectsMap[projectID]

	if strings.TrimSpace(projectList.pathToInstaller) == "" {
		utils.LogDebug("Skipping invocation of CLI command due to no installer path.")
		return

	}

	if !exists || value == nil {
		utils.LogSevere("Asked to invoke CLI on a project that wasn't in the projects map: " + projectID)
		return
	}

	value.cliState.OnFileChangeEvent()

}

/** Generate an overview of the state of the project list, including the projects being watched. */
func (projectList *ProjectList) handleRequestDebugMsg(projectsMap map[string]*projectObject) string {
	result := ""
	for projectID, obj := range projectsMap {

		if obj == nil {
			continue
		}

		result += "- " + projectID + " -> " + obj.project.PathToMonitor
		if obj.eventBatchUtil != nil {
			result += " | " + strings.TrimSpace(obj.eventBatchUtil.RequestDebugMessage())
		}

		if len(obj.project.IgnoredPaths) > 0 {

			result += " | ignoredPaths: "

			for _, val := range obj.project.IgnoredPaths {
				result += "'" + val + "' "
			}
		}

		result += "\n"

	}
	return result

}

/**
 * This function processes data that is from the GET API response; we use this to synchronize the list of projects
 * that we are watching with what the server wants us to watch.  */
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

		watchService.RemoveRootPath(fileToMonitor, *(removedProject.project))
	}

	// Next, create new projects, or updating existing ones
	for _, project := range *entries {
		projectList.processProject(project, projectsMap, postOutputQueue, watchService)
	}

}

/**
 * This function processes data that is from the WebSocket endpoint event; we use this to add/remove/update the list of projects
 * that we are watching with what the server wants us to watch.
 *
 * The difference between 'update from GET' and 'update from WebSocket' is that 'update from GET' does not indicate
 * how the project list changes, whereas 'update from WebSocket' does via the 'ChangeType'
 */
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
					watchService.RemoveRootPath(pathToRemove, projectFromWS)
				} else {
					utils.LogSevere("Watch service is not set in project list and a RemoveRootPath was missed: " + pathToRemove)
				}
			}

		} else {
			projectList.processProject(projectFromWS, projectsMap, postOutputQueue, watchService)
		}
	}

}

/**
 * Synchronize the project in our projectsMap (if it exists), with the new 'projectToProcess' from the server.
 * If it doesn't exist, create it.*/
func (projectList *ProjectList) processProject(projectToProcess models.ProjectToWatch, projectsMap map[string]*projectObject, postOutputQueue *HttpPostOutputQueue, watchService *WatchService) {

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
				projectToProcess.ChangeType = "" // TODO: the only non-immutable line
				currProjWatchState.project = &projectToProcess

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

		currProjWatchState, err := projectList.newProjectObject(projectToProcess, postOutputQueue)
		if err != nil {
			utils.LogSevereErr("Error on creation of new project object", err)
			return
		}
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

/** This function is called with a new file change entry, which is filtered (if necessary) then patched to the project's batch utility object.  */
func handleReceiveNewWatchEventEntries(projectMatch *models.ProjectToWatch, entry *models.WatchEventEntry, projectsMap map[string]*projectObject) {

	utils.LogDebug("Received new watch entry: " + entry.EventType + " " + entry.Path + " " + projectMatch.ProjectID)

	filter, err := utils.NewPathFilter(projectMatch)
	if err != nil {
		utils.LogSevere("Could not create filter for " + projectMatch.ProjectID)
		return
	}

	path := utils.ConvertAbsolutePathWithUnixSeparatorsToProjectRelativePath(entry.Path, projectMatch.PathToMonitor)

	if path == nil || len(*path) == 0 {
		return
	}

	if projectMatch.IgnoredPaths != nil {

		if filter.IsFilteredOutByPath(*path) {
			utils.LogDebug("Filtered out '" + *path + "' due to path filter")
			return
		} else {
			// Apply the path filter against parent paths as well (if path is /a/b/c, then also try to match against /a/b and /a)
			pathsToProcess := utils.SplitRelativeProjectPathIntoComponentPaths(*path)
			for _, val := range pathsToProcess {
				if filter.IsFilteredOutByPath(val) {
					return
				}
			}
		}

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

/**
 * Information maintained for each project that is being monitored by the
 * watcher. This includes information on what to watch/filter (the
 * ProjectToWatch), the batch util (one batch util object exists per project),
 * and which watch service (internal/external) is being used for this project.
 */
type projectObject struct {
	project        *models.ProjectToWatch
	eventBatchUtil *FileChangeEventBatchUtil
	cliState       *CLIState // Nullable
}

func (projectList *ProjectList) newProjectObject(project models.ProjectToWatch, postOutputQueue *HttpPostOutputQueue) (*projectObject, error) {

	var cliState *CLIState
	var err error

	if strings.TrimSpace(projectList.pathToInstaller) == "" {
		cliState = nil

	} else {

		// Here we convert the path to an absolute, canonical OS path for use by cwctl
		var path string
		path, err = utils.ConvertAbsoluteUnixStyleNormalizedPathToLocalFile(project.PathToMonitor)

		if err != nil {
			return nil, err
		}

		cliState, err = NewCLIState(project.ProjectID, projectList.pathToInstaller, path)
		if err != nil {
			return nil, err
		}

	}

	return &projectObject{
		&project,
		NewFileChangeEventBatchUtil(project.ProjectID, postOutputQueue, cliState),
		cliState, // May be null
	}, nil
}
