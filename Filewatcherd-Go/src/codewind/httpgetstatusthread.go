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
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type HttpGetStatusThread struct {
	refreshStatusChan chan interface{}
	baseURL           string
}

func (hg *HttpGetStatusThread) SignalStatusRefreshNeeded() {
	utils.LogDebug("SignalStatusRefreshNeeded called.")
	hg.refreshStatusChan <- nil
}

func NewHttpGetStatusThread(baseURL string, projectList *ProjectList) (*HttpGetStatusThread, error) {

	baseURL = utils.StripTrailingForwardSlash(baseURL)

	if !utils.IsValidURLBase(baseURL) {
		return nil, errors.New("URL is invalid: " + baseURL)
	}

	reconnectNeeded := make(chan interface{})

	result := &HttpGetStatusThread{
		reconnectNeeded,
		baseURL,
	}

	go runGetStatusThread(result, projectList)

	result.SignalStatusRefreshNeeded()

	// Every 120 seconds, refresh the status
	ticker := time.NewTicker(120 * time.Second)
	go func() {
		for {
			<-ticker.C
			utils.LogDebug("GetStatus ticker ticked.")
			result.SignalStatusRefreshNeeded()
		}
	}()

	return result, nil

}

func runGetStatusThread(data *HttpGetStatusThread, projectList *ProjectList) {
	utils.LogInfo("Http GET status thread started.")

	backoff := utils.NewExponentialBackoff()

	for {
		<-data.refreshStatusChan

		// Once a refresh status request is issued, keep trying until it succeeds.
		success := false
		for !success {

			err := doGetRequest(data.baseURL, backoff.GetFailureDelay(), projectList)
			if err != nil {
				utils.LogErrorErr("Error from GET request", err)
				backoff.SleepAfterFail()
				backoff.FailIncrease()
			} else {
				backoff.SuccessReset()
				success = true
			}
		}

		// On success, drain the channel of any other requests that occurred during this time.
		channelEmpty := false
		for !channelEmpty {
			select {
			case <-data.refreshStatusChan:
			default:
				channelEmpty = true
			}
		}

		utils.LogDebug("GET request successfully sent and received.")

	} // end for
}

func doGetRequest(baseURL string, failureDelay int, projectList *ProjectList) error {

	// Wait before issuing a request, due to a previous failed request
	if failureDelay > 0 {
		time.Sleep(time.Duration(failureDelay) * time.Millisecond)
	}
	result, err := sendGet(baseURL)

	if err != nil {
		return err
	}

	if result != nil {
		projectList.UpdateProjectListFromGetRequest(result)
	}

	return nil

}

func sendGet(baseURL string) (*models.WatchlistEntries, error) {

	url := baseURL + "/api/v1/projects/watchlist"

	utils.LogInfo("Initiating GET request to " + url)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	resp, err := client.Get(url)
	if err != nil || resp == nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errMsg := "Get response failed for " + url + ", response code: " + strconv.Itoa(resp.StatusCode)
		utils.LogError(errMsg)
		return nil, errors.New(errMsg)
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil || body == nil {
		utils.LogError("Get response failed for " + url + ", unable to read body")
		return nil, err
	}

	// Strip EOL characters to ensure it fits on one log line.
	bodyStr := string(body)
	bodyStr = strings.ReplaceAll(bodyStr, "\r", "")
	bodyStr = strings.ReplaceAll(bodyStr, "\n", "")

	utils.LogInfo("GET request completed, for " + url + ". Response: " + bodyStr)

	var entries models.WatchlistEntryList
	err = json.Unmarshal(body, &entries)
	if err != nil {
		utils.LogError("Get response failed for" + url + ", unable to unmarshal body.")
		return nil, err
	}

	return &entries.Projects, nil
}
