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
	"os"
	"time"
)

/* This is the entrypoint for the application.
 * The application takes one optional argument, which is the URL of the Codewind server. */
func main() {

	// Default URL if no args
	baseURL := "http://localhost:9090"

	var installerPath string

	// If one arg is specified, use it as a URL
	if len(os.Args) >= 2 {
		baseURL = os.Args[1]

		if len(os.Args) == 3 {
			installerPath = os.Args[2]
		}
	}

	baseURL = utils.StripTrailingForwardSlash(baseURL)

	httpPostOutputQueue, err := NewHttpPostOutputQueue(baseURL)
	if err != nil {
		utils.LogSevereErr("Unable to create HTTP POST output queue", err)
		return
	}

	projectList := NewProjectList(httpPostOutputQueue, installerPath)

	clientUUID := *utils.GenerateUuid()

	watchService := NewWatchService(projectList, baseURL, clientUUID)

	projectList.SetWatchService(watchService)

	httpGetStatusThread, err := NewHttpGetStatusThread(baseURL, projectList)

	if err != nil {
		utils.LogSevereErr("Unable to create HTTP GET status thread", err)
		return
	}

	StartWSConnectionManager(baseURL, projectList, httpGetStatusThread)

	debugTimer := NewDebugTimer(watchService, projectList, httpPostOutputQueue)
	debugTimer.Start()

	for {
		time.Sleep(1000 * time.Millisecond)
	}
}
