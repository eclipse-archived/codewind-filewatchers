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
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

/*
 * The purpose of this is to call the cwctl project sync command, in order to allow the
 * Codewind CLI to detect and communicate file changes to the server.
 *
 * This class will ensure that only one instance of the cwctl project sync command is running
 * at a time, per project.
 *
 * For automated testing, if the `MOCK_CWCTL_INSTALLER_PATH` environment variable is specified, a mock cwctl command
 * written in Java (as a runnable JAR) can be used to test this class.
 */
type CLIState struct {
	projectID string

	installerPath string

	projectPath string

	/** For automated testing only */
	mockInstallerPath string

	channel chan CLIStateChannelEntry
}

func NewCLIState(projectIDParam string, installerPathParam string, projectPathParam string) (*CLIState, error) {

	if installerPathParam == "" {
		// This object should not be instantiated if the installerPath is empty.
		return nil, errors.New("Installer path is empty: " + installerPathParam)
	}

	result := &CLIState{
		projectID:         projectIDParam,
		installerPath:     installerPathParam,
		projectPath:       projectPathParam,
		mockInstallerPath: strings.TrimSpace(os.Getenv("MOCK_CWCTL_INSTALLER_PATH")),
		channel:           make(chan CLIStateChannelEntry),
	}

	go result.readChannel()

	return result, nil

}

func (state *CLIState) OnFileChangeEvent() error {

	if strings.TrimSpace(state.projectPath) == "" {
		msg := "Project path passed to CLIState is empty, so ignoring file change event."
		utils.LogSevere(msg)
		return errors.New(msg)
	}

	// Inform channel that a new file change list was received (but don't actually send it)
	state.channel <- CLIStateChannelEntry{nil}

	return nil
}

func (state *CLIState) readChannel() {
	processWaiting := false // Once the current command completes, should we start another one
	processActive := false  // Is there currently a cwctl command active.

	var lastTimestamp int64 = 0

	for {

		channelResult := <-state.channel

		if channelResult.runProjectReturn != nil {
			// Event: Previous run of cwctl command has completed
			processActive = false

			rpr := channelResult.runProjectReturn

			if rpr.errorCode == 0 {
				// Success, so update the tiemstamp to the process start time.
				lastTimestamp = rpr.spawnTime
				utils.LogInfo("Updating timestamp to latest: " + strconv.FormatInt(lastTimestamp, 10))

			} else {
				utils.LogSevere("Non-zero error code from installer: " + rpr.output)
			}

		} else {
			// Another thread has informed us of new file changes
			processWaiting = true
		}

		if !processActive && processWaiting {
			// Start a new process if there isn't one running, and we received an update event.
			processWaiting = false
			processActive = true
			go state.runProjectCommand(lastTimestamp)
		}
	}

}

/* If non-null, then is a runProjectCommand response, otherwise, is a new file change. */
type CLIStateChannelEntry struct {
	runProjectReturn *RunProjectReturn
}

func (state *CLIState) runProjectCommand(timestamp int64) {

	firstArg := ""

	currInstallPath := state.installerPath

	var args []string

	adjustedTimestampInMsecs := (time.Now().UnixNano() / int64(time.Millisecond)) - timestamp

	if state.mockInstallerPath == "" {
		firstArg = state.installerPath
		// Example:
		// cwctl project sync -p
		// /Users/tobes/workspaces/git/eclipse/codewind/codewind-workspace/lib5 \
		// -i b1a78500-eaa5-11e9-b0c1-97c28a7e77c7 -t 12345

		// Do not wrap paths in quotes; it's not needed and Go doesn't like that :P

		args = append(args, "project", "sync", "-p", state.projectPath, "-i", state.projectID, "-t",
			strconv.FormatInt(adjustedTimestampInMsecs, 10))

	} else {
		firstArg = "java"
		// args = append(args, "java")

		args = append(args, "-jar", state.mockInstallerPath, "-p", state.projectPath, "-i",
			state.projectID, "-t", strconv.FormatInt(adjustedTimestampInMsecs, 10))

		currInstallPath = state.mockInstallerPath
	}

	debugStr := ""

	for _, key := range args {

		debugStr += "[ " + key + "] "
	}

	utils.LogInfo("Calling cwctl project sync with: [" + state.projectID + "] { " + debugStr + "}")

	// Start process and wait for complete on this thread.

	installerPwd := filepath.Dir(currInstallPath)

	spawnTimeInMsecs := (time.Now().UnixNano() / int64(time.Millisecond))

	cmd := exec.Command(firstArg, args...)
	cmd.Dir = installerPwd

	stdoutStderr, err := cmd.CombinedOutput()

	utils.LogInfo("Cwctl call completed, elapsed time of cwctl call: " + strconv.FormatInt((time.Now().UnixNano()/int64(time.Millisecond))-spawnTimeInMsecs, 10))

	if err != nil && err.(*exec.ExitError).ExitCode() != 0 {
		errorCode := err.(*exec.ExitError).ExitCode()
		utils.LogError("Error running 'project sync' installer command: " + debugStr)
		utils.LogError("Out: " + string(stdoutStderr))

		result := RunProjectReturn{
			errorCode,
			string(stdoutStderr),
			spawnTimeInMsecs,
		}

		state.channel <- CLIStateChannelEntry{&result}

	} else {

		utils.LogInfo("Successfully ran installer command: " + debugStr)
		utils.LogInfo("Output:" + string(stdoutStderr)) // TODO: Convert to DEBUG once everything matures.

		result := RunProjectReturn{
			0,
			string(stdoutStderr),
			spawnTimeInMsecs,
		}

		state.channel <- CLIStateChannelEntry{&result}

	}
}

/** Return value of runProjectCommand(). */
type RunProjectReturn struct {
	errorCode int
	output    string
	spawnTime int64
}
