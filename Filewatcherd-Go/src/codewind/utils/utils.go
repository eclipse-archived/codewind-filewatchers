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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type ExponentialBackoff struct {
	MinFailureDelay int

	FailureDelay int

	MaxFailureDelay int

	BackoffExponent float32
}

func (b *ExponentialBackoff) GetFailureDelay() int {
	return b.FailureDelay
}

func NewExponentialBackoff() ExponentialBackoff {
	return ExponentialBackoff{
		200,
		0,
		4000,
		1.5,
	}
}

func (b *ExponentialBackoff) SleepAfterFail() {

	if b.FailureDelay == 0 {
		b.FailureDelay = b.MinFailureDelay
	}

	time.Sleep(time.Duration(b.FailureDelay) * time.Millisecond)
}

func (b *ExponentialBackoff) FailIncrease() {

	if b.FailureDelay == 0 {
		b.FailureDelay = b.MinFailureDelay
		return
	}

	b.FailureDelay = int(float32(b.FailureDelay) * b.BackoffExponent)
	if b.FailureDelay > b.MaxFailureDelay {
		b.FailureDelay = b.MaxFailureDelay
	}

}

func (b *ExponentialBackoff) SuccessReset() {
	b.FailureDelay = 0
}

func StripTrailingForwardSlash(str string) string {

	for strings.HasSuffix(str, "/") {

		str = str[:len(str)-1]
	}

	return str

}

func IsValidURLBase(str string) bool {

	if strings.HasPrefix(str, "http://") {
		return true
	}

	if strings.HasPrefix(str, "https://") {
		return true
	}

	return false
}

func GenerateUuid() *string {
	u := make([]byte, 16)
	_, err := rand.Read(u)
	if err != nil {
		return nil
	}

	result := hex.EncodeToString(u)

	return &result
}

func FormatTime(t time.Time) string {

	formatted := "[" + fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d.%03d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(), (t.Nanosecond()/1000000)) + "]"

	return formatted

}
