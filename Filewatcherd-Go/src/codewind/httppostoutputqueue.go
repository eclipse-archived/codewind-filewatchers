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

type HttpPostOutputQueue struct {
	url              string
	workInputChannel chan *PostQueueChannelMessage
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
	}

	// Start the work manager goroutine
	go workManagerNew(result)

	return result, nil
}

func (postQueue *HttpPostOutputQueue) AddToQueue(projectIDParam string, timestamp int64, base64Compressed []string) {

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

	postQueue.workInputChannel <- &PostQueueChannelMessage{
		chunkGroup,
	}

	utils.LogDebug("Added file changes to queue: " + strconv.Itoa(len(base64Compressed)) + " " + projectIDParam)

}

func workManagerNew(queue *HttpPostOutputQueue) {

	utils.LogInfo("HttpPostOutputQueue thread has started for" + queue.url)

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

			activeWorkers = queueMoreWorkIfNeeded(priorityList, activeWorkers, MaxWorkers, &backoff, workCompleteChannel, queue)

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

			activeWorkers = queueMoreWorkIfNeeded(priorityList, activeWorkers, MaxWorkers, &backoff, workCompleteChannel, queue)

		}
	}

}

func queueMoreWorkIfNeeded(priorityList *ChunkGroupPriorityList, currActiveWorkers int, MaxWorkers int, backoff *utils.ExponentialBackoff, workCompleteChannel chan *PostQueueWorkResultChannel, queue *HttpPostOutputQueue) int {

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
			go doRequest(chunk, workCompleteChannel, backoff.GetFailureDelay(), queue)
			currActiveWorkers++
		} else {
			break
		}

	}

	return currActiveWorkers

}

func doRequest(work *PostQueueChunk, workCompleteChannel chan *PostQueueWorkResultChannel, failureDelay int, queue *HttpPostOutputQueue) {

	// Wait before issuing a request, due to a previous failed request
	if failureDelay > 0 {
		time.Sleep(time.Duration(failureDelay) * time.Millisecond)
	}
	err := sendPost(queue, work)

	utils.LogDebug("sendPost complete")

	if err != nil {
		utils.LogErrorErr("Error occurred on send: ", err)

		workCompleteChannel <- &PostQueueWorkResultChannel{work, false}

	} else {

		workCompleteChannel <- &PostQueueWorkResultChannel{work, true}
	}

	utils.LogDebug("Work signaled on workCompleteChannel in HTTP post queue")
}

func sendPost(queue *HttpPostOutputQueue, chunk *PostQueueChunk) error {

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
