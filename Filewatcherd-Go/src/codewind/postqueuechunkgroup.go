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

/**
 * The 'chunk group' maintains the list of chunks for a specific timestamp that
 * we are currently trying to send to the server.
 *
 * Each chunk in the chunk group is in one of these states:
 *
 * - AVAILABLE_TO_SEND: Chunks in this state are available to be sent by the next available worker.
 * - WAITING_FOR_ACK: Chunks in this state are in the process of being sent by one of the workers.
 * - COMPLETE: Chunks in this state have been sent and acknowledged by the server.
 *
 * The primary goal of chunk groups is to ensure that chunks will never be sent
 * to the server out of ascending-timestamp order: eg we will never send the
 * server a chunk of 'timestamp 20', then a chunk of 'timestamp 19'. The
 * 'timestamp 20' chunks will wait for all of the 'timestamp 19' chunks to be
 * sent.
 *
 * This class is thread safe.
 */
type PostQueueChunkGroup struct {
	chunkMap          map[int] /*chunk id -> */ *PostQueueChunk
	chunkStatus       map[int] /*chunk id -> */ ChunkStatus
	timestamp         int64
	expireTimeInNanos int64
}

/**
 * A large number of file changes will be split into 'bite-sized pieces' called
 * chunks. Each chunk communicates a subset of the full change list, and is
 * communicated on a separate HTTP POST request.
 *
 * Instances of this class are immutable.
 */
type PostQueueChunk struct {
	/** The ID of a chunk will be 1 <= id <= chunkTotal */
	chunkID int
	/** The total # of chunks that will e sent for this project id and timestamp. */
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
