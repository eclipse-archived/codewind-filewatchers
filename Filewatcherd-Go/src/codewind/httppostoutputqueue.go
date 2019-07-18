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
	"errors"
	"net/http"
	"strconv"

	"codewind/utils"
	"crypto/tls"
	"time"
)

/**
 * This class is responsible for informing the server (via HTTP post request) of
 * any file/directory changes that have occurred.
 *
 * The eventbatchutil.go functions (indirectly) calls this code with a list of
 * base-64+compressed strings (containing the list of changes), and then this
 * code breaks the changes down into small chunks and sends them in the body of
 * individual HTTP POST requests.
 */
type HttpPostOutputQueue struct {
	url                 string
	workInputChannel    chan *PostQueueChannelMessage
	requestDebugChannel chan chan string
}

type PostQueueChannelMessage struct {
	chunkGroup *PostQueueChunkGroup
}

type PostQueueWorkResultChannel struct {
	chunk   *PostQueueChunk
	success bool
}

func NewHttpPostOutputQueue(url string) (*HttpPostOutputQueue, error) {

	url = utils.StripTrailingForwardSlash(url)

	if !utils.IsValidURLBase(url) {
		return nil, errors.New("URL is invalid: " + url)
	}

	workChannel := make(chan *PostQueueChannelMessage)

	result := &HttpPostOutputQueue{
		url,
		workChannel,
		make(chan chan string),
	}

	// Start the work manager goroutine
	go result.workManager()

	return result, nil
}

func (queue *HttpPostOutputQueue) AddToQueue(projectIDParam string, timestamp int64, base64Compressed []string) {

	chunkGroup := &PostQueueChunkGroup{
		make(map[int]*PostQueueChunk, 0),
		make(map[int]ChunkStatus, 0),
		timestamp,
	}

	for index, base64String := range base64Compressed {

		chunk := &PostQueueChunk{
			index + 1,
			len(base64Compressed),
			base64String,
			projectIDParam,
			timestamp,
			chunkGroup,
		}

		chunkGroup.chunkMap[chunk.chunkID] = chunk
		chunkGroup.chunkStatus[chunk.chunkID] = AVAILABLE_TO_SEND

	}

	queue.workInputChannel <- &PostQueueChannelMessage{
		chunkGroup,
	}

	utils.LogDebug("Added file changes to queue: " + strconv.Itoa(len(base64Compressed)) + " " + projectIDParam)

}

func (queue *HttpPostOutputQueue) RequestDebugMessage() chan string {
	result := make(chan string)

	queue.requestDebugChannel <- result

	return result
}

func (queue *HttpPostOutputQueue) workManager() {

	utils.LogInfo("HttpPostOutputQueue thread has started for " + queue.url)

	const MaxWorkers int = 3

	priorityList := NewChunkGroupPriorityList()

	workCompleteChannel := make(chan *PostQueueWorkResultChannel)

	// The number of threads currently handling HTTP request
	activeWorkers := 0

	backoff := utils.NewExponentialBackoff()

	for {

		select {
		case newWork := <-queue.workInputChannel:

			// If we received a new work item, push it on the queue
			priorityList.AddToList(newWork.chunkGroup)
			utils.LogDebug("Added new work to HttpPostOutputQueue")

			activeWorkers = queue.queueMoreWorkIfNeeded(priorityList, activeWorkers, MaxWorkers, &backoff, workCompleteChannel)

		case completedWork := <-workCompleteChannel:
			activeWorkers--
			if completedWork.success == true {
				backoff.SuccessReset()

				completedWork.chunk.parent.InformChunkSent(completedWork.chunk)

			} else {
				utils.LogDebug("Existing work failed, so requeueing to HttpPostOutputQueue")
				backoff.FailIncrease()

				completedWork.chunk.parent.InformChunkFailedToSend(completedWork.chunk)

			}

			activeWorkers = queue.queueMoreWorkIfNeeded(priorityList, activeWorkers, MaxWorkers, &backoff, workCompleteChannel)

		case debugResponseChannel := <-queue.requestDebugChannel:
			result := "- active-workers: " + strconv.Itoa(activeWorkers)

			chunkGroupList := priorityList.GetList()

			result += "  chunkGroupList-size: " + strconv.Itoa(len(chunkGroupList)) + "\n"
			if len(chunkGroupList) > 0 {
				result += "\n"
				result += "- HTTP Post Chunk Group List:\n"
				for _, val := range chunkGroupList {

					projectID := ""
					if len(val.chunkMap) > 0 {
						pqc, exists := val.chunkMap[0]
						if exists && pqc != nil {
							projectID = pqc.projectID
						}
					}
					result += "  - projectID: " + projectID + "  timestamp: " + strconv.FormatInt(val.timestamp, 10) + "\n"

				}
			}
			debugResponseChannel <- result
		}
	}

}

/** Start a new goroutine to send POST requests, if there is more work available and we are not at max workers. */
func (queue *HttpPostOutputQueue) queueMoreWorkIfNeeded(priorityList *ChunkGroupPriorityList, currActiveWorkers int, MaxWorkers int, backoff *utils.ExponentialBackoff, workCompleteChannel chan *PostQueueWorkResultChannel) int {

	// While there is both more work available, and at least one free worker to do the work
	for priorityList.Len() > 0 && currActiveWorkers < MaxWorkers {

		// Remove the chunk group at the front of the list if we have sent all its packets
		chunkGroup := priorityList.Peek()
		if chunkGroup.IsGroupComplete() {
			priorityList.Pop()
			continue
		}

		// If there is at least one chunk waiting to send, send it
		chunk := chunkGroup.AcquireNextChunkAvailableToSend()
		if chunk != nil {
			go queue.doRequest(chunk, workCompleteChannel, backoff.GetFailureDelay())
			currActiveWorkers++
		} else {
			break
		}

	}

	return currActiveWorkers

}

/** Call 'sendPost' then communicate the result back on the repsonse channel. */
func (queue *HttpPostOutputQueue) doRequest(work *PostQueueChunk, workCompleteChannel chan *PostQueueWorkResultChannel, failureDelay int) {

	// Wait before issuing a request, due to a previous failed request
	if failureDelay > 0 {
		time.Sleep(time.Duration(failureDelay) * time.Millisecond)
	}
	err := queue.sendPost(work)

	utils.LogDebug("sendPost complete")

	if err != nil {
		utils.LogErrorErr("Error occurred on send: ", err)

		workCompleteChannel <- &PostQueueWorkResultChannel{work, false}

	} else {

		workCompleteChannel <- &PostQueueWorkResultChannel{work, true}
	}

	utils.LogDebug("Work signaled on workCompleteChannel in HTTP post queue")
}

/** Construct and send the HTTP POST request, and return an error on either failure or !200 */
func (queue *HttpPostOutputQueue) sendPost(chunk *PostQueueChunk) error {

	buffer := bytes.NewBufferString("{\"msg\" : \"" + chunk.base64Compressed + "\"}")

	url := queue.url + "/api/v1/projects/" + chunk.projectID + "/file-changes?timestamp=" + strconv.FormatInt(chunk.timestamp, 10) + "&chunk=" + strconv.FormatInt((int64)(chunk.chunkID), 10) + "&chunk_total=" + strconv.FormatInt((int64)(chunk.chunkTotal), 10)

	utils.LogInfo("Sending POST request to " + url + " with payload size " + strconv.Itoa(buffer.Len()))

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	resp, err := client.Post(url, "application/json", buffer)
	if err != nil {
		return err
	}

	if resp == nil {
		return errors.New("Response was nil")
	} else if resp.StatusCode != 200 {
		return errors.New("Response code was != 200")
	}

	defer resp.Body.Close()

	return nil
}
