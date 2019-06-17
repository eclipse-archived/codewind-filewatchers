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

type WatchService struct {
	watchServiceChannel chan *WatchServiceChannelMessage
	clientUUID          string
}

/** Only one of the fields of this struct should be non-nil per instance */
type WatchServiceChannelMessage struct {
	addOrRemove         *AddRemoveRootPathChannelMessage
	directoryWaitResult *WatchDirectoryWaitResultMessage
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

func (service *WatchService) AddRootPath(path string, projectFromWS *models.ProjectToWatch) {

	debugStr := "Add " + projectFromWS.ProjectID + " @" + time.Now().String()

	msg := &AddRemoveRootPathChannelMessage{
		true,
		path,
		projectFromWS,
		debugStr,
	}

	msgPackage := &WatchServiceChannelMessage{
		msg,
		nil,
	}

	service.watchServiceChannel <- msgPackage

}

func (service *WatchService) RemoveRootPath(path string, projectFromWS *models.ProjectToWatch) {
	debugStr := "Remove " + projectFromWS.ProjectID + " @" + time.Now().String()
	msg := &AddRemoveRootPathChannelMessage{
		false,
		path,
		projectFromWS,
		debugStr,
	}

	msgPackage := &WatchServiceChannelMessage{
		msg,
		nil,
	}

	service.watchServiceChannel <- msgPackage
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
					informWatchSuccessStatus(msg.project, false, baseURL, publicObject)
				}

			}
		}

	}

}

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
		&sync.Mutex{},
		make(map[string]bool),
		make(map[string]bool),
	}

	watchedProjects[project.ProjectID] = watcher

	go waitForWatchedPathSuccess(addMsg.path, project, service)

}

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

	informWatchSuccessStatus(project, success, baseURL, service)

}

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

	utils.LogInfo("waitForWatchedPathSuccess completed with status of watchSuccess: " + strconv.FormatBool(watchSuccess))

	result := &WatchDirectoryWaitResultMessage{
		path,
		projectToWatch,
		watchSuccess,
	}

	msgPackage := &WatchServiceChannelMessage{
		nil,
		result,
	}

	watchService.watchServiceChannel <- msgPackage
}

func closeWatcherIfNeeded(existing *CodewindWatcher) {

	var watcherToClose *fsnotify.Watcher

	existing.lock.Lock()
	if !existing.closed_synch_lock {
		watcherToClose = existing.fsnotifyWatcher
		existing.closed_synch_lock = true
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

type CodewindWatcher struct {
	fsnotifyWatcher   *fsnotify.Watcher
	rootPath          string
	id                string
	closed_synch_lock bool
	lock              *sync.Mutex
	watchedDirMap     map[string] /*path*/ bool
	isDirMap          map[string] /*path*/ bool /** The last time we saw this existing, was it a file or a dir*/
}

func startWatcher(cWatcher *CodewindWatcher, path string, projectList *ProjectList, service *WatchService, project *models.ProjectToWatch) error {

	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		return err
	}

	cWatcher.fsnotifyWatcher = watcher

	go func() {

		watcherFuncID := strconv.FormatUint(rand.Uint64(), 10)

		for {
			select {
			case event, ok := <-watcher.Events:

				utils.LogDebug("Raw: " + event.Name + " " + event.Op.String() + ", id: " + cWatcher.id + ", watcher func id: " + watcherFuncID)

				if !ok {

					cWatcher.lock.Lock()
					isClosed := cWatcher.closed_synch_lock
					cWatcher.lock.Unlock()

					if isClosed {
						utils.LogInfo("Ignoring a !ok that was received after the watcher was closed.")

						// Exit the channel read function, here
						return
					} else {
						utils.LogSevere("!ok from watcher: " + event.Name + " " + event.Op.String() + " " + event.String() + " " + cWatcher.id)
						continue
					}
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
								// closeWatcherIfNeeded(cWatcher)
								// service.RemoveRootPath(event.Name, project)
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
					utils.LogInfo("Ignoring an error or !ok that was received after the watcher was closed.")
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

			}
		}
	}()

	addedFiles, addedDirs, walkErr := walkPathAndAdd(path, cWatcher)

	if walkErr != nil {
		return walkErr
	}

	utils.LogInfo("Initial path walk complete for " + path + ", addedFiles: " + strconv.Itoa(len(addedFiles)) + ", addedDirs: " + strconv.Itoa(len(addedDirs)))

	return nil

}

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
			// utils.LogDebug("files in path " + path + " - " + strconv.Itoa(len(files)))
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

func informWatchSuccessStatus(ptw *models.ProjectToWatch, success bool, baseURL string, service *WatchService) {

	go func() {
		successVal := "true"

		backoffUtil := utils.NewExponentialBackoff()

		if !success {
			successVal = "false"
		}
		buffer := bytes.NewBufferString("{\"success\" : \"" + successVal + "\"}")

		url := baseURL + "/api/v1/projects/" + ptw.ProjectID + "/file-changes/" + ptw.ProjectWatchStateID + "/status?clientUuid=" + service.clientUUID

		passed := false

		for !passed {
			utils.LogDebug("Sending PUT request to " + url)

			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
				
			client := &http.Client{Transport: tr}

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

		utils.LogInfo("Successfully informed server of watch state for " + ptw.ProjectID + " watch-state-id: " + ptw.ProjectWatchStateID + ", success: " + successVal)

	}()

}
