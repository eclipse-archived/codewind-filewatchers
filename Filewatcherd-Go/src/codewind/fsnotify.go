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
	"bytes"
	"codewind/models"
	"codewind/utils"
	"crypto/tls"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

/**
 * The WatchService class uses the directory/file monitoring functionality of the 3rd party
 * fsnotify go library for file monitoring.
 *
 * (Add/Remove)RootPath are called by the ProjectList goroutine whenever a new directory needs to be
 * monitored, or no longer needs to be monitored, or the filters have changed.
 *
 * These requests are converted into channel messages and passed to the internal goroutine to ensure
 * thread safety.
 */
type WatchService struct {
	watchServiceChannel chan *WatchServiceChannelMessage
	clientUUID          string
}

/** Only one of the fields of this struct should be non-nil per instance */
type WatchServiceChannelMessage struct {
	addOrRemove         *AddRemoveRootPathChannelMessage
	directoryWaitResult *WatchDirectoryWaitResultMessage
	debugMessage        *FsNotifyDebugMessage
}

type FsNotifyDebugMessage struct {
	responseChannel chan string
}

type AddRemoveRootPathChannelMessage struct {
	isAdd   bool
	path    string
	project *models.ProjectToWatch
	debug   string
}

type WatchDirectoryWaitResultMessage struct {
	path    string
	project *models.ProjectToWatch
	success bool
}

func NewWatchService(projectList *ProjectList, baseUrl string, clientUUID string) *WatchService {

	result := &WatchService{
		make(chan *WatchServiceChannelMessage),
		clientUUID,
	}

	go watchServiceEventLoop(result, projectList, baseUrl)

	return result
}

func (service *WatchService) AddRootPath(path string, projectFromWS models.ProjectToWatch) {

	debugStr := "Add " + projectFromWS.ProjectID + " @" + time.Now().String()

	msg := &AddRemoveRootPathChannelMessage{
		true,
		path,
		&projectFromWS,
		debugStr,
	}

	msgPackage := &WatchServiceChannelMessage{
		addOrRemove:         msg,
		directoryWaitResult: nil,
		debugMessage:        nil,
	}

	service.watchServiceChannel <- msgPackage

}

func (service *WatchService) RemoveRootPath(path string, projectFromWS models.ProjectToWatch) {
	debugStr := "Remove " + projectFromWS.ProjectID + " @" + time.Now().String()
	msg := &AddRemoveRootPathChannelMessage{
		false,
		path,
		&projectFromWS,
		debugStr,
	}

	msgPackage := &WatchServiceChannelMessage{
		addOrRemove:         msg,
		directoryWaitResult: nil,
		debugMessage:        nil,
	}

	service.watchServiceChannel <- msgPackage
}

func (service *WatchService) RequestDebugMessage() chan string {
	responseChannel := make(chan string)

	msgPackage := &WatchServiceChannelMessage{
		debugMessage: &FsNotifyDebugMessage{responseChannel},
	}

	service.watchServiceChannel <- msgPackage

	return responseChannel
}

func watchServiceEventLoop(publicObject *WatchService, projectList *ProjectList, baseURL string) {

	/* key: project ID */
	watchedProjects := make(map[string]*CodewindWatcher)

	for {

		select {
		case watchServiceMessage := <-publicObject.watchServiceChannel:

			// If we are receiving an add/remove from our public API
			if watchServiceMessage.addOrRemove != nil {
				addOrRemoveRootPathMsg := watchServiceMessage.addOrRemove
				utils.LogInfo("Processing message: " + addOrRemoveRootPathMsg.debug)

				if addOrRemoveRootPathMsg.isAdd {
					addRootPathInternal_step1(addOrRemoveRootPathMsg, watchedProjects, projectList, baseURL, publicObject)
				} else {
					removeRootPathInternal(addOrRemoveRootPathMsg, watchedProjects)
				}
			}

			// If a path we have previously added is reported as either succeeding or failing
			if watchServiceMessage.directoryWaitResult != nil {
				msg := watchServiceMessage.directoryWaitResult

				utils.LogInfo("Processing directory wait result message: " + msg.path + " " + msg.project.ProjectID + " " + strconv.FormatBool(msg.success))

				if msg.success {
					addRootPathInternal_step2(msg.path, msg.project, watchedProjects, projectList, baseURL, publicObject)
				} else {
					informWatchSuccessStatus(msg.project, false, baseURL, publicObject, projectList)
				}

			}

			// If we receive a debug request, respond with the current status
			if watchServiceMessage.debugMessage != nil {
				responseChannel := watchServiceMessage.debugMessage.responseChannel

				result := ""
				utils.LogInfo("Processing debug message")

				for key, val := range watchedProjects {
					result += "- " + key + " | " + val.rootPath + " | "
					if val.closed_synch_lock {
						result += "(closed)"
					} else {
						result += "(open)"
					}
					result += "\n"

					// If the watcher is open, then request debug data from it.
					val.lock.Lock()
					if val.open_synch_lock {
						result += val.latest_debug_state_lock
					}
					val.lock.Unlock()

				}

				responseChannel <- result
			}
		}

	}

}

/**
 * First, we construct the CodewindWatcher in an unopened state, then start a new goroutine to wait for the project directory
 * to exist. */
func addRootPathInternal_step1(addMsg *AddRemoveRootPathChannelMessage, watchedProjects map[string]*CodewindWatcher, projectList *ProjectList,
	baseURL string, service *WatchService) {

	project := addMsg.project
	projectID := project.ProjectID
	utils.LogInfo("Starting to add root path " + addMsg.path + " for project " + projectID)

	existing, exists := watchedProjects[projectID]
	if exists {
		// If the watcher exists in the map, then close it so we can start a new one
		closeWatcherIfNeeded(existing)
	}

	watcher := &CodewindWatcher{
		nil,
		addMsg.path,
		strconv.FormatUint(rand.Uint64(), 10),
		false,
		false,
		"",
		&sync.Mutex{},
		make(map[string]bool),
		make(map[string]bool),
	}

	watchedProjects[project.ProjectID] = watcher

	go waitForWatchedPathSuccess(addMsg.path, project, service)

}

/** This function is called once the project directory exists, so we can now start the fsnotify watcher and report success */
func addRootPathInternal_step2(path string, project *models.ProjectToWatch, watchedProjects map[string]*CodewindWatcher, projectList *ProjectList,
	baseURL string, service *WatchService) {

	cWatcher, exists := watchedProjects[project.ProjectID]
	if !exists {
		// If it doesn't exist, it has already been removed
		return
	}

	err := startWatcher(cWatcher, path, projectList, service, project)

	success := true

	if err != nil {
		utils.LogErrorErr("Error on establishing watch", err)
		success = false
	}

	informWatchSuccessStatus(project, success, baseURL, service, projectList)

}

/**
 * Wait up to X minutes for the project directory to exist; if it succeeds proceed to step 2, otherwise
 * report an error back to the server. */
func waitForWatchedPathSuccess(path string, projectToWatch *models.ProjectToWatch, watchService *WatchService) {
	expireTime := time.Now().Add(time.Minute * 5)

	var nextOutputTime *time.Time

	watchSuccess := false

	for {

		if statVal, err := os.Stat(path); err == nil {

			if statVal.IsDir() {
				watchSuccess = true
				break
			}

		} else {

			if nextOutputTime == nil {
				nextOutput := time.Now().Add(time.Second * 10)
				nextOutputTime = &nextOutput
			} else if time.Now().After(*nextOutputTime) {
				nextOutputTime = nil
				utils.LogInfo("Waiting for " + path + " to exist")
			}

			time.Sleep(100 * time.Millisecond)
		}

		if time.Now().After(expireTime) {
			watchSuccess = false
			break
		}
	}

	utils.LogInfo("waitForWatchedPathSuccess completed for projId " + projectToWatch.ProjectID + " with status of watchSuccess: " + strconv.FormatBool(watchSuccess))

	result := &WatchDirectoryWaitResultMessage{
		path,
		projectToWatch,
		watchSuccess,
	}

	msgPackage := &WatchServiceChannelMessage{
		addOrRemove:         nil,
		directoryWaitResult: result,
		debugMessage:        nil,
	}

	watchService.watchServiceChannel <- msgPackage
}

/** Close an old watcher, either because the project is no longer being watched, or the filters have been updated. */
func closeWatcherIfNeeded(existing *CodewindWatcher) {

	var watcherToClose *fsnotify.Watcher

	existing.lock.Lock()
	if !existing.closed_synch_lock {
		watcherToClose = existing.fsnotifyWatcher
		existing.latest_debug_state_lock = ""
		existing.closed_synch_lock = true
		existing.open_synch_lock = false
		utils.LogInfo("Existing watcher found, so deleting old watcher " + existing.rootPath)
	} else {
		utils.LogSevere("A closed entry should not exist in the watcher map.")
	}
	existing.lock.Unlock()

	// Close the watcher outside the lock
	if watcherToClose != nil {
		err := watcherToClose.Close()
		if err != nil {
			utils.LogSevereErr("Error on closing watcher", err)
		}
	}
}

func removeRootPathInternal(removeMsg *AddRemoveRootPathChannelMessage, watchedProjects map[string]*CodewindWatcher) {

	projectID := removeMsg.project.ProjectID

	existing, exists := watchedProjects[projectID]
	if exists {
		utils.LogInfo("Removing project " + projectID + " with root path " + removeMsg.path)
		closeWatcherIfNeeded(existing)
		delete(watchedProjects, projectID)
	} else {
		utils.LogError("Attempted to remove project " + projectID + " with root path " + removeMsg.path + " but it was not found in watchedPaths")
	}

}

/** Only immutable objects (id, rootPath), or lockable objects, should be accessed across threads */
type CodewindWatcher struct {
	fsnotifyWatcher *fsnotify.Watcher /* reference to notify api */
	rootPath        string            /* root directory of the path*/
	id              string

	/* whether the watcher is closed, and therefore events can be ignored, lock on  */
	closed_synch_lock bool

	/* whether the watcher is open, and therefore the project directory exists */
	open_synch_lock bool

	/* every X minutes, the state of the watcher is stored in this string, for thread-safe use by the debug thread. */
	latest_debug_state_lock string

	/** Acquire this before reading/writing any of the above _lock variables. */
	lock *sync.Mutex

	/** A list of all the paths we have added to fsnotifyWatcher */
	watchedDirMap map[string] /*path -> */ bool

	/** The last time we saw this existing, was it a file or a dir; used to handle directory deletion case*/
	isDirMap map[string] /*path -> is directory */ bool
}

/** Do an initial directory scan to add the new project directory, and kick off the goroutine to handle watcher events.  */
func startWatcher(cWatcher *CodewindWatcher, path string, projectList *ProjectList, service *WatchService, project *models.ProjectToWatch) error {

	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		return err
	}

	cWatcher.lock.Lock()
	cWatcher.open_synch_lock = true
	cWatcher.lock.Unlock()

	cWatcher.fsnotifyWatcher = watcher

	go func() {

		watcherFuncID := strconv.FormatUint(rand.Uint64(), 10)

		debugUpdateTimer := time.NewTicker(10 * time.Minute)

		for {
			select {
			case event, ok := <-watcher.Events:

				if utils.IsLogDebug() {
					utils.LogDebug("Raw fsnotify event: " + event.Name + " " + event.Op.String() + ", id: " + cWatcher.id + ", watcher func id: " + watcherFuncID + " watch state Id: " + project.ProjectWatchStateID)
				}

				if !ok {

					cWatcher.lock.Lock()
					isClosed := cWatcher.closed_synch_lock
					cWatcher.lock.Unlock()

					if isClosed {
						utils.LogDebug("Ignoring a !ok that was received after the watcher was closed.")
						// Exit the channel read function, here
						return
					} else {
						utils.LogSevere("!ok from watcher while the watcher was still open: " + event.Name + " " + event.Op.String() + " " + event.String() + " " + cWatcher.id)
						continue
					}
				}

				cWatcher.lock.Lock()
				isClosed := cWatcher.closed_synch_lock
				cWatcher.lock.Unlock()
				if isClosed {
					utils.LogDebug("Ignoring event on closed watcher: " + event.Name + " " + event.Op.String())
					continue
				}

				changeType := ""
				isDir := false

				fileExists := false

				stat, err := os.Stat(event.Name)

				if err != nil {
					fileExists = false

					// File doesn't exist, so check our map of old paths
					isDirMapVal, exists := cWatcher.isDirMap[event.Name]
					if exists {
						// fmt.Println("Exists in map: " + event.Name + " w/ val " + strconv.FormatBool(isDirMapVal))
						// This is required for the delete directory case: a deleted directory cannot be stat-ed
						isDir = isDirMapVal
					} else {
						// fmt.Println("Does not exist map: " + event.Name)
					}

				} else {
					fileExists = true

					if stat.IsDir() {
						// If it exists, and it's a directory
						isDir = true
					}

				}

				watchEventEntries := make([]*models.WatchEventEntry, 0)

				if isDir {
					// If is directory CREATE/DELETE, then we need to start/stop watching it
					if event.Op&fsnotify.Create == fsnotify.Create {
						utils.LogDebug("Adding new directory watch: " + event.Name)
						newFilesFound, newDirsFound, err := walkPathAndAdd(event.Name, cWatcher)
						if err != nil {
							utils.LogSevereErr("Unexpected error from file walk: "+event.Name, err)
						} else {

							// For any files that were found in new directories, create CREATE entries for them.
							for _, val := range newFilesFound {
								newEvent, err := newWatchEventEntry("CREATE", val, false)
								cWatcher.isDirMap[val] = false

								if err == nil {
									watchEventEntries = append(watchEventEntries, newEvent)
								} else {
									utils.LogSevereErr("Unexpected watch event entry error", err)
								}

							}

							for _, val := range newDirsFound {
								newEvent, err := newWatchEventEntry("CREATE", val, true)
								cWatcher.isDirMap[val] = true

								if err == nil {
									watchEventEntries = append(watchEventEntries, newEvent)
								} else {
									utils.LogSevereErr("Unexpected watch event entry error", err)
								}

							}

						}
						changeType = "CREATE"
					} else if event.Op&fsnotify.Remove == fsnotify.Remove {
						utils.LogDebug("Removing directory watch: " + event.Name)
						watcher.Remove(event.Name)
						delete(cWatcher.watchedDirMap, event.Name)
						changeType = "DELETE"

						// If the directory being removed is the project directory itself, then stop the watcher
						if event.Name == cWatcher.rootPath {

							if fileExists {
								utils.LogSevere("The watch service has nothing to watch, but the root file still exists. This shouldn't happen. Path: " + event.Name)
							} else {
								utils.LogInfo("REMOVED - The watch service has nothing to watch, so the watcher is stopping:" + event.Name)
							}

						}
					} else {
						utils.LogDebug("Ignoring: " + event.Name)
					}
				} else {

					// Files
					if event.Op&fsnotify.Create == fsnotify.Create {
						changeType = "CREATE"
					} else if event.Op&fsnotify.Write == fsnotify.Write {
						changeType = "MODIFY"
					} else if event.Op&fsnotify.Remove == fsnotify.Remove {
						changeType = "DELETE"
					}
				}

				if len(watchEventEntries) > 0 {
					for _, val := range watchEventEntries {
						utils.LogDebug("WatchEventEntry (dir): " + val.EventType + " " + val.Path + " " + strconv.FormatBool(val.IsDir))
						projectList.ReceiveNewWatchEventEntries(val, project)
					}
				}

				if changeType != "" {
					newEvent, err := newWatchEventEntry(changeType, event.Name, isDir)

					if changeType != "DELETE" {
						cWatcher.isDirMap[event.Name] = isDir
					}
					if err != nil {
						utils.LogSevereErr("Unexpected file path conversion error", err)
					} else {
						utils.LogDebug("WatchEventEntry: " + changeType + " " + event.Name + " " + strconv.FormatBool(isDir) + " " + cWatcher.id)
						projectList.ReceiveNewWatchEventEntries(newEvent, project)
					}
				}
			case err, ok := <-watcher.Errors:

				cWatcher.lock.Lock()
				isClosed := cWatcher.closed_synch_lock
				cWatcher.lock.Unlock()

				if isClosed {
					if err != nil {
						utils.LogInfo("Ignoring an error or !ok that was received after the watcher was closed, for project " + project.ProjectID + ": " + err.Error())
					} else {
						utils.LogInfo("Ignoring an error or !ok that was received after the watcher was closed, for project " + project.ProjectID)
					}

					// Exit the channel read function, here
					return
				}

				if err != nil {
					utils.LogSevereErr("Watcher error, ok: "+strconv.FormatBool(ok), err)
				} else {
					utils.LogSevere("Watcher error received, ok: " + strconv.FormatBool(ok))
				}
				if !ok {
					continue
				}

			case _ = <-debugUpdateTimer.C: // Update the internal debug state every X minutes

				// Print the first X paths in 'watchedDirMap'
				count := 0
				result := ""
				for key := range cWatcher.watchedDirMap {
					result += "  - " + key + "\n"
					count++

					if count >= 20 {
						break
					}
				}

				cWatcher.lock.Lock()
				if !cWatcher.closed_synch_lock { // Only update if still open
					cWatcher.latest_debug_state_lock = result
				}
				cWatcher.lock.Unlock()

			}
		} // end for
	}() // end go func

	addedFiles, addedDirs, walkErr := walkPathAndAdd(path, cWatcher)

	if walkErr != nil {
		return walkErr
	}

	utils.LogInfo("Initial path walk complete for " + path + ", addedFiles: " + strconv.Itoa(len(addedFiles)) + ", addedDirs: " + strconv.Itoa(len(addedDirs)))

	return nil

}

/** Begin to recursively scan pathParam */
func walkPathAndAdd(pathParam string, cWatcher *CodewindWatcher) ([]string, []string, error) {
	utils.LogDebug("Beginning to walk path " + pathParam)

	newFilesFound := make([]string, 0)
	newDirsFound := make([]string, 0)

	// For every directory under (and inclusive of) pathParam:
	// - Add a watch for the directory
	// - List the files in the directory and add them as new changes to report
	// - Based on handling inotify race conditions, described here: https://lwn.net/Articles/605128/

	walkErr := walkPathAndAddInternal(pathParam, cWatcher, &newFilesFound, &newDirsFound)

	if walkErr != nil {
		utils.LogDebug("Path walk complete for " + pathParam + ", with error")

		return nil, nil, walkErr
	}
	utils.LogDebug("Path walk complete for " + pathParam + ".")
	return newFilesFound, newDirsFound, nil
}

/**
 * Recursively scan pathParam, and add a new fsnotify watch for the path if it isn't already watched.
 * For any files found in the directory, add them to newFilesFound (as these need to be CREATE entries) */
func walkPathAndAddInternal(path string, cWatcher *CodewindWatcher, newFilesFound *[]string, newDirsFound *[]string) error {
	_, exists := cWatcher.watchedDirMap[path]

	if !exists {
		strList := make([]string, 0)
		strList = append(strList, path)

		cWatcher.watchedDirMap[path] = true
		err := cWatcher.fsnotifyWatcher.Add(path)
		utils.LogDebug("Added watch: " + path)
		if err != nil {
			utils.LogSevereErr("Unable to walk path: "+path, err)
		}

		*newDirsFound = append(*newDirsFound, path)
		files, err := ioutil.ReadDir(path)
		if err != nil {
			utils.LogSevereErr("Unable to read directory: "+path, err)
		} else {
			// For each of the files in the directory, add them to 'new files found' array, otherwise recurse
			for _, f := range files {

				val := path + string(os.PathSeparator) + f.Name()
				if !f.IsDir() {
					*newFilesFound = append(*newFilesFound, val)
				} else {
					walkPathAndAddInternal(val, cWatcher, newFilesFound, newDirsFound)
				}

			}
		}
	}

	return nil
}

func newWatchEventEntry(eventType string, path string, isDir bool) (*models.WatchEventEntry, error) {
	path = strings.ReplaceAll(path, "\\", "/")
	path = utils.ConvertFromWindowsDriveLetter(path)

	path, err := utils.NormalizeDriveLetter(path)

	if err != nil {
		return nil, err
	}

	return &models.WatchEventEntry{
		EventType: eventType,
		Path:      path,
		IsDir:     isDir,
	}, nil
}

/** Start a new goroutine to communicate to the server the success/failure of the initial watch. */
func informWatchSuccessStatus(ptw *models.ProjectToWatch, success bool, baseURL string, service *WatchService, projectList *ProjectList) {

	go func() {

		if success {
			// Inform the CLI on watch success, if needed
			projectList.CLIFileChangeUpdate(ptw.ProjectID)
		}

		successVal := "true"

		backoffUtil := utils.NewExponentialBackoff()

		if !success {
			successVal = "false"
		}

		url := baseURL + "/api/v1/projects/" + ptw.ProjectID + "/file-changes/" + ptw.ProjectWatchStateID + "/status?clientUuid=" + service.clientUUID

		passed := false

		for !passed {
			utils.LogDebug("Sending PUT request to " + url)

			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}

			client := &http.Client{Transport: tr}

			buffer := bytes.NewBufferString("{\"success\" : \"" + successVal + "\"}")
			req, err := http.NewRequest(http.MethodPut, url, buffer)

			req.Header.Set("Content-Type", "application/json")

			if err != nil {
				utils.LogErrorErr("Error from PUT request ", err)
				backoffUtil.SleepAfterFail()
				backoffUtil.FailIncrease()
				passed = false
				continue
			}

			resp, err := client.Do(req)
			if err != nil {
				utils.LogErrorErr("Error from PUT request ", err)
				backoffUtil.SleepAfterFail()
				backoffUtil.FailIncrease()
				passed = false
				continue
			}

			if resp.StatusCode != 200 {
				utils.LogError("Status code request from PUT was not 200 - " + strconv.Itoa(resp.StatusCode))
				backoffUtil.SleepAfterFail()
				backoffUtil.FailIncrease()
				passed = false
				continue
			}

			passed = true

		}

		utils.LogInfo("Successfully informed server of watch state for " + ptw.ProjectID + ", watch-state-id: " + ptw.ProjectWatchStateID + ", success: " + successVal)

	}()

}
