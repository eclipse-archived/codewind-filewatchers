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
	"strings"
	"time"
)

/**
 * Every X minutes, this timer thread will run and output the internal state of
 * each of the internal components of the filewatcher. This should run
 * infrequently, as it can be fairly verbose (eg every 30 minutes).
 *
 * The goal of this timer is to identify bugs/performance issues that might only
 * be detectable when the workbench is open for long periods of time (memory
 * leaks, resources we aren't closing, etc).
 */
type DebugTimer struct {
	watchService    *WatchService
	projectList     *ProjectList
	postOutputQueue *HttpPostOutputQueue
}

func NewDebugTimer(watchService *WatchService, projectList *ProjectList, postOutputQueue *HttpPostOutputQueue) *DebugTimer {
	result := &DebugTimer{
		watchService,
		projectList,
		postOutputQueue,
	}

	return result
}

/** Start (or restart) the timer */
func (debugTimer *DebugTimer) Start() {

	// This is intentionally a timer, and not a ticker.
	timer := time.NewTimer(30 * time.Minute)
	go func() {
		for range timer.C {
			debugTimer.OutputDebug()
			return // Exit the goroutine after one invocation, to terminate the thread/channel
		}
	}()
}

func (debugTimer *DebugTimer) OutputDebug() {

	result := ""

	result += "---------------------------------------------------------------------------------------\n\n"

	watchServiceResult := <-debugTimer.watchService.RequestDebugMessage()
	result += "WatchService:\n" + strings.TrimSpace(watchServiceResult) + "\n\n"

	result += "Project List:\n" + strings.TrimSpace(<-debugTimer.projectList.RequestDebugMessage()) + "\n\n"

	result += "HTTP Post Output Queue:\n" + strings.TrimSpace(<-debugTimer.postOutputQueue.RequestDebugMessage()) + "\n\n"

	result += "---------------------------------------------------------------------------------------\n"

	for _, val := range strings.Split(result, "\n") {

		utils.LogInfo("[status] " + val)
	}

	// Restart the timer
	debugTimer.Start()
}
