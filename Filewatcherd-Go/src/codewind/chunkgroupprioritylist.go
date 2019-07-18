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
	"sort"
)

/** ChunkGroupPriorityList maintains a pseudo priority queue of chunk groups, sorted ascending by chunk group timestamp */
type ChunkGroupPriorityList struct {
	list []*PostQueueChunkGroup
}

func NewChunkGroupPriorityList() *ChunkGroupPriorityList {

	list := make([]*PostQueueChunkGroup, 0)

	return &ChunkGroupPriorityList{
		list,
	}
}

func (listObj *ChunkGroupPriorityList) GetList() []*PostQueueChunkGroup {
	return listObj.list
}

func (listObj *ChunkGroupPriorityList) AddToList(newItem *PostQueueChunkGroup) {
	listObj.list = append(listObj.list, newItem)
	listObj.sortList()
}

func (listObj *ChunkGroupPriorityList) sortList() {

	if listObj.Len() < 2 {
		return
	}

	sort.SliceStable(listObj.list, func(i, j int) bool {
		// Sort ascending by timestamp (confirmed)
		return (listObj.list)[i].timestamp < (listObj.list)[j].timestamp
	})

}

func (listObj *ChunkGroupPriorityList) Len() int {
	return len(listObj.list)
}

func (listObj *ChunkGroupPriorityList) Peek() *PostQueueChunkGroup {
	if listObj.Len() == 0 {
		return nil
	}

	return listObj.list[0]
}

func (listObj *ChunkGroupPriorityList) Pop() *PostQueueChunkGroup {
	if listObj.Len() == 0 {
		return nil
	}

	result := listObj.list[0]

	listObj.list = listObj.list[1:]

	return result
}
