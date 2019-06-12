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
	"codewind/utils"
)

type ChunkStatus int

const (
	AVAILABLE_TO_SEND ChunkStatus = iota + 1
	WAITING_FOR_ACK
	COMPLETE
)

type PostQueueChunkGroup struct {
	chunkMap    map[int] /*chunk id*/ *PostQueueChunk
	chunkStatus map[int] /*chunk id*/ ChunkStatus
	timestamp   int64
}

type PostQueueChunk struct {
	chunkID          int
	chunkTotal       int
	base64Compressed string
	projectID        string
	timestamp        int64
	parent           *PostQueueChunkGroup
}

func (pqcg *PostQueueChunkGroup) IsGroupComplete() bool {

	for _, val := range pqcg.chunkStatus {

		if val != COMPLETE {
			return false
		}

	}

	return true
}

func (pqcg *PostQueueChunkGroup) AcquireNextChunkAvailableToSend() *PostQueueChunk {

	match := -1

	for key, val := range pqcg.chunkStatus {

		if val == AVAILABLE_TO_SEND {
			match = key
			break
		}
	}

	if match == -1 {
		return nil
	}

	pqcg.chunkStatus[match] = WAITING_FOR_ACK

	return pqcg.chunkMap[match]
}

func (pqcg *PostQueueChunkGroup) InformChunkSent(chunk *PostQueueChunk) {

	currStatus := pqcg.chunkStatus[chunk.chunkID]
	if currStatus != WAITING_FOR_ACK {
		utils.LogSevere("Unexpected status of chunk, should be WAITING")
	}

	pqcg.chunkStatus[chunk.chunkID] = COMPLETE

}

func (pqcg *PostQueueChunkGroup) InformChunkFailedToSend(chunk *PostQueueChunk) {

	currStatus := pqcg.chunkStatus[chunk.chunkID]
	if currStatus != WAITING_FOR_ACK {
		utils.LogSevere("Unexpected status of chunk, should be WAITING")
	}

	pqcg.chunkStatus[chunk.chunkID] = AVAILABLE_TO_SEND

}
