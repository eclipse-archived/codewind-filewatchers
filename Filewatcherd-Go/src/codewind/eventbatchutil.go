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
	"codewind/utils"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

/**
 * When file/directory change events occur in quick succession (within
 * milliseconds of each other), it tends to imply that they are related. For
 * example, if you were refactoring Java code, that refactoring operation might
 * touch a bunch of source files at once. This means, in order to avoid
 * performing extra builds, we should batch together changes that occur within
 * close temporal proximity.
 *
 * However, we don't want to wait TOO long for new events, otherwise this
 * introduces latency between when the user makes a change, and when their build
 * actually starts.
 *
 * FileChangeEventBatchUtil implements an algorithm that groups together changes that occur
 * within X milliseconds of each other.
 *
 * The algorithm is: After at least one event is received, wait for there to be
 * be no more events in the stream of events (within eg 1000 msecs) before
 * sending them to the server. If an event is seen within 1000 msecs, the timer
 * is reset and a new 1000 msec timer begins. Batch together events seen since
 * within a given timeframe, and send them as a single request.
 *
 * This code receives file change events from the watch service, and forwards
 * batched groups of events to the HTTP POST output queue.
 *
 */
type FileChangeEventBatchUtil struct {
	filesChangesChan      chan []ChangedFileEntry
	debugState_synch_lock string    // Lock 'lock' before reading/writing this
	cliState              *CLIState // nullable
	lock                  *sync.Mutex
}

func NewFileChangeEventBatchUtil(projectID string, postOutputQueue *HttpPostOutputQueue, cliStateParam *CLIState) *FileChangeEventBatchUtil {

	result := &FileChangeEventBatchUtil{
		filesChangesChan:      make(chan []ChangedFileEntry),
		debugState_synch_lock: "",
		lock:                  &sync.Mutex{},
		cliState:              cliStateParam,
	}

	go result.fileChangeListener(projectID, postOutputQueue)

	return result
}

func (e *FileChangeEventBatchUtil) AddChangedFiles(changedFileEntries []ChangedFileEntry) {
	e.filesChangesChan <- changedFileEntries
}

func (e *FileChangeEventBatchUtil) RequestDebugMessage() string {

	e.lock.Lock()
	defer e.lock.Unlock()

	return e.debugState_synch_lock
}

func (e *FileChangeEventBatchUtil) fileChangeListener(projectID string, postOutputQueue *HttpPostOutputQueue) {

	utils.LogInfo("EventBatchUtil listener started for " + projectID)

	eventsReceivedSinceLastBatch := []ChangedFileEntry{}

	timerChan := make(chan *time.Timer)

	debugTimeSinceLastFileChange := time.Now()

	debugTimeSinceLastTimerReceived := time.Now()

	var timer1 *time.Timer

	for {

		select {
		case timerReceived := <-timerChan:

			// First, update our debug stats
			debugTimeSinceLastTimerReceived = time.Now()
			e.updateDebugState(debugTimeSinceLastFileChange, debugTimeSinceLastTimerReceived)

			// Only process a timer elapsed event if the event is for the timer that is currently active (prevent race condition)
			if timer1 != nil && timer1 == timerReceived {

				if len(eventsReceivedSinceLastBatch) > 0 {
					processAndSendEvents(eventsReceivedSinceLastBatch, projectID, postOutputQueue, e.cliState)
				}
				eventsReceivedSinceLastBatch = []ChangedFileEntry{}
				timer1 = nil
			}

		case receivedFileChanges := <-e.filesChangesChan:
			debugTimeSinceLastFileChange = time.Now()
			e.updateDebugState(debugTimeSinceLastFileChange, debugTimeSinceLastTimerReceived)

			eventsReceivedSinceLastBatch = append(eventsReceivedSinceLastBatch, receivedFileChanges...)
			if timer1 != nil {
				timer1.Stop()
			}
			timer1 = time.NewTimer(1000 * time.Millisecond)
			go func(t *time.Timer) {
				<-t.C
				// If timer is still active, send an elapsed time
				// if t == timer1 {
				timerChan <- t
				// }
			}(timer1)
		}

	} // end for

}

func (e *FileChangeEventBatchUtil) updateDebugState(debugTimeSinceLastFileChange time.Time, debugTimeSinceLastTimerReceived time.Time) {
	result := "lastFileChangeSeen: " + utils.FormatTime(debugTimeSinceLastFileChange)
	result += "   timeSinceLastTimer: " + utils.FormatTime(debugTimeSinceLastTimerReceived) + "\n"

	e.lock.Lock()
	e.debugState_synch_lock = result
	e.lock.Unlock()
}

/** Process the event list, split it into chunks, then pass it to the HTTP POST output queue */
func processAndSendEvents(eventsToSend []ChangedFileEntry, projectID string, postOutputQueue *HttpPostOutputQueue, cliState *CLIState) {
	sort.SliceStable(eventsToSend, func(i, j int) bool {

		// Sort ascending by timestamp
		return eventsToSend[i].timestamp < eventsToSend[j].timestamp

	})

	// Remove any contiguous create/delete events
	eventsToSend = removeDuplicateEventsOfType(eventsToSend, "CREATE")
	eventsToSend = removeDuplicateEventsOfType(eventsToSend, "DELETE")

	if len(eventsToSend) == 0 {
		return
	}

	// Split the entries into requests (chunks), to ensure that each request is no larger
	// then a given size.
	mostRecentTimestamp := eventsToSend[len(eventsToSend)-1]

	changeSummary := generateChangeListSummaryForDebug(eventsToSend)
	utils.LogInfo(
		"Batch change summary for " + projectID + "@ " + strconv.FormatInt(mostRecentTimestamp.timestamp, 10) + ": " + changeSummary)

	if cliState != nil {
		// Inform CLI of changes
		cliState.OnFileChangeEvent()

	} else {
		// Use the old way of communicating file changes via POST packets.

		// TODO: Remove this entire else block once CWCTL sync is mature.

		var fileListsToSend [][]changedFileEntryJSON

		for len(eventsToSend) > 0 {

			var currList []changedFileEntryJSON

			// Remove at most X paths from currList
			for len(currList) < 625 && len(eventsToSend) > 0 {
				cfe := eventsToSend[0]
				eventsToSend = eventsToSend[1:]
				currList = append(currList, *cfe.toJSON())

			}

			if len(currList) > 0 {
				fileListsToSend = append(fileListsToSend, currList)
			}
		}

		var stringsToSend []string

		for _, jsonArray := range fileListsToSend {
			jaString, err := json.Marshal(jsonArray)

			if err != nil {
				utils.LogSevere("Unable to marshal JSON")
				continue
			}

			compressedStr, err := compressAndConvertString(jaString)
			if err != nil {
				// We shouldn't ever get an error from compressing or conversion
				utils.LogSevere("Unable to compress JSON")
				continue
			}

			stringsToSend = append(stringsToSend, *compressedStr)
		}

		// Pass the list of chunks to the HTTP Post output queue, for transmission to the server
		utils.LogDebug("Strings to send " + strconv.Itoa(len(stringsToSend)))
		if len(stringsToSend) > 0 {
			postOutputQueue.AddToQueue(projectID, mostRecentTimestamp.timestamp, stringsToSend)
		}

	}

}

func generateChangeListSummaryForDebug(eventsToSend []ChangedFileEntry) string {
	result := "[ "

	for _, val := range eventsToSend {

		if val.eventType == "CREATE" {
			result += "+"
		} else if val.eventType == "MODIFY" {
			result += ">"
		} else if val.eventType == "DELETE" {
			result += "-"
		} else {
			result += "?" // if you see this, it's a bug ;)
		}

		filename := val.path
		index := strings.LastIndex(filename, "/")
		if index != -1 {
			filename = filename[(index + 1):]
		}

		result += filename + " "

		// Only output the first 256 chars of changes, to reduce log verbosity
		if len(result) > 256 {
			break
		}
	}

	if len(result) > 256 {
		result += " (...) "
	}

	result += "]"

	return result
}

/** For any given path: If there are multiple entries of the same type in a row, then remove all but the first. */
func removeDuplicateEventsOfType(entries []ChangedFileEntry, changeType string) []ChangedFileEntry {

	if changeType == "MODIFY" {
		utils.LogSevere("Unsupported event type: MODIFY")
		return entries
	}

	/* path -> value not used */
	containsPath := make(map[string]bool)

	for x := 0; x < len(entries); x++ {
		cfe := entries[x]

		path := cfe.path

		if cfe.eventType == changeType {
			_, exists := containsPath[path]
			if exists {
				utils.LogDebug("Removing duplicate event: " + cfe.toDebugString())
				entries = append(entries[:x], entries[x+1:]...)
				x--
			} else {
				containsPath[path] = true
			}

		} else {
			delete(containsPath, path)
		}

	}

	return entries
}

func compressAndConvertString(strBytes []byte) (*string, error) {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	_, err := w.Write(strBytes)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}

	toBase64 := base64.StdEncoding.EncodeToString(b.Bytes())

	return &toBase64, nil
}

/**
 * Simple representation of a single change: the file/dir path that changed,
 * what type of change, and when. These are then consumed by the batch
 * processing utility.
 */
type ChangedFileEntry struct {
	path      string
	eventType string
	timestamp int64
	directory bool
}

type changedFileEntryJSON struct {
	Path      string `json:"path"`
	Timestamp int64  `json:"timestamp"`
	Type      string `json:"type"`
	Directory bool   `json:"directory"`
}

func (e *ChangedFileEntry) toJSON() *changedFileEntryJSON {

	return &changedFileEntryJSON{
		e.path,
		e.timestamp,
		e.eventType,
		e.directory,
	}
}

func (e *ChangedFileEntry) toDebugString() string {

	return e.path + " " + strconv.FormatInt(e.timestamp, 10) + " " + e.eventType + " " + strconv.FormatBool(e.directory)
}

func NewChangedFileEntry(path string, eventType string, timestamp int64, directory bool) (*ChangedFileEntry, error) {

	if len(strings.TrimSpace(path)) == 0 || len(strings.TrimSpace(eventType)) == 0 || timestamp <= 0 {
		return nil, errors.New("Invalid changed entry value: " + path + " " + eventType + " " + strconv.FormatInt(timestamp, 10))
	}

	return &ChangedFileEntry{
		path,
		eventType,
		timestamp,
		directory,
	}, nil

}
