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
	"codewind/models"
	"codewind/utils"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

/**
 * The purpose of the WebSocket Connection Manager is to initiate and maintain the WebSocket
 * connection between the filewatcher and the server.
 *
 * After queueEstablishConnection(...) is called, we will keep trying to connect
 * to the server until it succeeds. If that connection ever goes down for any
 * reason, queueEstablishConnection() still start the reconnection process over
 * again.
 *
 * This class also sends a simple "keep alive" packet every X seconds (eg 25).
 */

type ReconnectMessage int

const (
	Reconnect = iota + 1
	Terminate
)

func StartWSConnectionManager(baseURL string, projectList *ProjectList, httpGetStatusThread *HttpGetStatusThread) error {
	baseURL = utils.StripTrailingForwardSlash(baseURL)

	if !utils.IsValidURLBase(baseURL) {
		return errors.New("URL is invalid: " + baseURL)
	}

	wsURLType := "ws"
	if strings.HasPrefix(baseURL, "https:") {
		wsURLType = "wss"
	}

	lastSlash := strings.LastIndex(baseURL, "/")
	if lastSlash == -1 {
		return errors.New("Invalid URL format, no slash found: " + baseURL)
	}

	hostnameAndPort := baseURL[lastSlash+1:]

	go eventLoop(wsURLType, hostnameAndPort, projectList, httpGetStatusThread)

	return nil
}

func eventLoop(wsURLType string, hostnameAndPort string, projectList *ProjectList, httpGetStatusThread *HttpGetStatusThread) {

	for {

		reconnectNeeded := make(chan ReconnectMessage)

		// Kick off websocket using channel
		startWebSocketThread(wsURLType, hostnameAndPort, reconnectNeeded, projectList, httpGetStatusThread)

		// We only read the first message from this channel, to avoid duplicates
		v := <-reconnectNeeded

		if v == Reconnect {
			// Ignore and loop to top
			utils.LogInfo("WebSocket thread received reconnect message.")

			// We lost the WebSocket connection, and theoretically might have missed
			// a watch refresh, so reacquire the latest watches.
			httpGetStatusThread.SignalStatusRefreshNeeded()

		} else if v == Terminate {
			utils.LogInfo("WebSocket thread received terminate message.")
			return
		}
	}

}

func startWebSocketThread(wsURLType string, hostnameAndPort string, triggerRetry chan ReconnectMessage, projectList *ProjectList, httpGetStatusThread *HttpGetStatusThread) {

	u := url.URL{Scheme: wsURLType, Host: hostnameAndPort, Path: "/websockets/file-changes/v1"}

	backoff := utils.NewExponentialBackoff()

	var c *websocket.Conn

	// Keep trying to connect on the WebSocket thread, until success
	for {

		utils.LogInfo("Connecting to " + u.String())

		dialer := &websocket.Dialer{}
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

		innerC, _, err := dialer.Dial(u.String(), nil)

		c = innerC

		if err != nil {
			utils.LogErrorErr("Error on connecting:", err)
			if innerC != nil {
				innerC.Close() // Unnecessary?
			}
		} else {
			// Success, so stop trying to connect
			break
		}

		// On failure, sleep
		backoff.SleepAfterFail()
		backoff.FailIncrease()
	}

	utils.LogInfo("Successfully connected to " + u.String())

	// On success, issue a GET request in case we missed anything.
	httpGetStatusThread.SignalStatusRefreshNeeded()

	ticker := time.NewTicker(25 * time.Second)
	tickerClosedChan := make(chan *time.Ticker)

	startWriteEmptyMessageTickerHandler(ticker, c, tickerClosedChan)

	c.SetCloseHandler(func(code int, text string) error {
		triggerRetry <- Reconnect
		utils.LogInfo("Close handler called with values: " + strconv.Itoa(code) + " " + text)

		if c != nil {
			c.Close()
		}

		ticker.Stop()
		tickerClosedChan <- ticker

		return nil
	})

	// Start a new listening thread, which informs us on failure
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				triggerRetry <- Reconnect
				utils.LogErrorErr("Read error:", err)
				c.Close()

				ticker.Stop()
				tickerClosedChan <- ticker
				return
			}

			var emptyInterface interface{}
			err = json.Unmarshal(message, &emptyInterface)
			m := emptyInterface.(map[string]interface{})
			if m["type"] == "debug" {
				// This string is sent only by automated tests
				if str, ok := m["msg"].(string); ok {
					utils.LogInfo("------------------------------------------------------------")
					utils.LogInfo("[Server-Debug] " + str)
					utils.LogInfo("------------------------------------------------------------")
				}
				continue
			}

			var watchChangeJSON models.WatchChangeJson
			error := json.Unmarshal(message, &watchChangeJSON)

			if error != nil {
				utils.LogSevereErr("Error occurred while unmarshalling JSON ", error)
				continue
			}

			projectUpdatesReceived := ""

			projectList.UpdateProjectListFromWebSocket(&watchChangeJSON)

			utils.LogInfo("Received watch change message from WebSocket: " + string(message))

			for x := 0; x < len(watchChangeJSON.Projects); x++ {

				entry := watchChangeJSON.Projects[x]
				projectUpdatesReceived += "[" + entry.ProjectID + " in " + entry.PathToMonitor + "], "
			}

			// Trim whitespace and trailing comma
			projectUpdatesReceived = strings.TrimSpace(projectUpdatesReceived)
			if strings.HasSuffix(projectUpdatesReceived, ",") {
				projectUpdatesReceived = projectUpdatesReceived[:len(projectUpdatesReceived)-1]
			}

			utils.LogInfo("Watch list change message received for { " + projectUpdatesReceived + " }")

		}
	}()

}

func startWriteEmptyMessageTickerHandler(ticker *time.Ticker, c *websocket.Conn, tickerClosedChan chan *time.Ticker) {

	// Start a new goroutine to send an empty json string every 25 seconds
	go func() {
		t := "{}"

		for {
			select {
			case <-ticker.C:
				utils.LogInfo("On ticker. writing to WebSocket...")
				// On ticker (every 25 seconds), send an empty string to the socket
				err := c.WriteMessage(websocket.TextMessage, []byte(t))
				if err != nil {
					utils.LogErrorErr("Unable to write empty WebSocket message", err)
					return
				}
			case <-tickerClosedChan:
				utils.LogInfo("Ticket channel is closed.")
				// If the ticker is closed, terminate the thread
				return
			}

		}
	}()

}
