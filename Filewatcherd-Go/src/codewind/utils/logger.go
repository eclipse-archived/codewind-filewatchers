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

package utils

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

type MonitorLogger struct {
	output   chan outputLine
	logLevel LogLevel
}

type outputLine struct {
	line      string
	err       bool
	timestamp int64
}

type LogLevel int

const (
	DEBUG  LogLevel = 1
	INFO   LogLevel = 2
	ERROR  LogLevel = 3
	SEVERE LogLevel = 4
)

var (
	logger *MonitorLogger
	once   sync.Once
)

func loggerInternal() *MonitorLogger {
	// Create a single instance of Logger, on first use
	once.Do(func() {
		messages := make(chan outputLine, 100)
		logger = &MonitorLogger{messages, INFO}
		go logger.logOutputter()
	})

	return logger
}

func LogDebug(msg string) {
	l := loggerInternal()

	if l.logLevel > DEBUG {
		return
	}
	l.out(msg)
}

func LogInfo(msg string) {
	l := loggerInternal()
	if l.logLevel > INFO {
		return
	}
	l.out(msg)

}

func LogError(msg string) {
	l := loggerInternal()
	if l.logLevel > ERROR {
		return
	}
	l.err("! ERROR !:" + msg)

}

func LogErrorErr(msg string, err error) {
	l := loggerInternal()
	if l.logLevel > ERROR {
		return
	}

	outputMsg := "! ERROR !: " + msg

	if err != nil {
		outputMsg += " - Error:" + err.Error()
	}

	l.err(outputMsg)
}

func LogSevere(msg string) {
	l := loggerInternal()
	l.err("!!! SEVERE !!!: " + msg)
}

func LogSevereErr(msg string, err error) {

	outputMsg := "!!! SEVERE !!!: " + msg

	if err != nil {
		outputMsg += " - Error:" + err.Error()
	}

	l := loggerInternal()
	l.err(outputMsg)
}

func (l *MonitorLogger) out(msg string) {
	l.output <- outputLine{
		msg,
		false,
		time.Now().UnixNano() / 1000000,
	}
}

func (l *MonitorLogger) err(msg string) {
	l.output <- outputLine{
		msg,
		true,
		time.Now().UnixNano() / 1000000,
	}
}

func (l *MonitorLogger) logOutputter() {

	startTime := time.Now()

	for {
		toPrint := <-l.output

		t := time.Now()
		formatted := "[" + fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d.%03d",
			t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second(), (t.Nanosecond()/1000000)) + "]"

		elapsedTimeInMsecs := toPrint.timestamp - ((startTime.UnixNano()) / 1000000)

		elapsedTimeInSeconds := int(elapsedTimeInMsecs / 1000)

		// Convert to 3-place decimal with padding
		elapsedTimeInDecimal := int(elapsedTimeInMsecs%1000) + 1000
		elapsedTimeInDecimalStr := strconv.Itoa(elapsedTimeInDecimal)[1:]

		time := formatted + " [" + strconv.Itoa(elapsedTimeInSeconds) + "." + elapsedTimeInDecimalStr + "] "

		if toPrint.err {
			os.Stderr.WriteString(time + toPrint.line + "\n")
		} else {
			os.Stdout.WriteString(time + toPrint.line + "\n")
		}
	}
}
