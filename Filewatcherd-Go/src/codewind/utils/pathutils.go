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
	"codewind/models"
	"errors"
	"regexp"
	"runtime"
	"strings"
	"unicode"
)

func IsWindowsAbsolutePath(absolutePath string) bool {

	if len(absolutePath) < 2 {
		return false
	}

	char0 := absolutePath[0]

	if !unicode.IsLetter(rune(char0)) {
		return false
	}

	if absolutePath[1] != ':' {
		return false
	}

	return true

}

/** Ensure that the drive is lowercase for Unix-style paths from Windows. */
func NormalizeDriveLetter(absolutePath string) (string, error) {

	if strings.Contains(absolutePath, "\\") {
		return "", errors.New("This function does not support Windows-style paths")
	}

	if len(absolutePath) < 2 {
		return absolutePath, nil
	}

	if !strings.HasPrefix(absolutePath, "/") {
		return "", errors.New("Path should begin with forward slash: " + absolutePath)
	}

	char0 := absolutePath[0]
	char1 := absolutePath[1]

	// Special case the absolute path of only 2 characters.
	if len(absolutePath) == 2 {

		if char0 == '/' && unicode.IsLetter(rune(char1)) && unicode.IsUpper(rune(char1)) {
			return "/" + string(unicode.ToLower(rune(char1))), nil
		} else {
			return absolutePath, nil
		}
	}

	char2 := absolutePath[2]
	if char0 == '/' && char2 == '/' && unicode.IsLetter(rune(char1)) && unicode.IsUpper(rune(char1)) {

		return "/" + string(unicode.ToLower(rune(char1))) + string(char2) + absolutePath[3:], nil

	} else {
		return absolutePath, nil
	}
}

/** C:\helloThere -> /c/helloThere */
func ConvertFromWindowsDriveLetter(absolutePath string) string {

	if !IsWindowsAbsolutePath(absolutePath) {
		return absolutePath
	}

	absolutePath = strings.ReplaceAll(absolutePath, "\\", "/")

	char0 := absolutePath[0]

	// Strip first two characters
	absolutePath = absolutePath[2:]

	absolutePath = "/" + string(unicode.ToLower(rune(char0))) + absolutePath

	return absolutePath

}

/** Same as below, but determine behaviour based on OS. */
func ConvertAbsoluteUnixStyleNormalizedPathToLocalFile(str string) (string, error) {

	if runtime.GOOS != "windows" {
		// For Mac/Linux, nothing to do
		return str, nil
	}

	return ConvertAbsoluteUnixStyleNormalizedPathToLocalFileOS(str, true)
}

/* Convert /c/Users/Administrator to c:\Users\Administrator */
func ConvertAbsoluteUnixStyleNormalizedPathToLocalFileOS(str string, isWindows bool) (string, error) {

	if !isWindows {
		return str, nil
	}

	if !strings.HasPrefix(str, "/") {
		return "", errors.New("Parameters must begin with slash")
	}

	if len(str) <= 1 {
		return "", errors.New("Cannot convert string with length of 0 or 1: " + str)
	}

	driveLetter := str[1]

	if !unicode.IsLetter(rune(driveLetter)) {
		return "", errors.New("Missing drive letter: " + str)
	}

	if len(str) == 2 {
		return string(driveLetter) + ":\\", nil
	}

	secondSlash := str[2]
	if secondSlash != '/' {
		return "", errors.New("Invalid path format: " + str)
	}

	return string(driveLetter) + ":\\" + strings.ReplaceAll(str[3:], "/", "\\"), nil

}

type PathFilter struct {
	filenameExcludePatterns []*regexp.Regexp
	pathExcludePatterns     []*regexp.Regexp
}

func NewPathFilter(project *models.ProjectToWatch) (*PathFilter, error) {

	result := PathFilter{
		make([]*regexp.Regexp, 0),
		make([]*regexp.Regexp, 0),
	}

	ignoredFilenames := project.IgnoredFilenames
	if ignoredFilenames != nil {
		for _, val := range ignoredFilenames {

			if strings.Contains(val, "/") || strings.Contains(val, "\\") {
				return nil, errors.New("Ignore filenames may not contain path separators: " + val)
			}

			text := strings.ReplaceAll(val, "*", ".*")
			re, err := regexp.Compile(text)
			if err != nil {
				LogSevere("Unable to compile regex: " + text)
				return nil, err
			}

			result.filenameExcludePatterns = append(result.filenameExcludePatterns, re)

		}
	}

	ignoredPaths := project.IgnoredPaths
	if ignoredPaths != nil {
		for _, val := range ignoredPaths {
			if strings.Contains(val, "\\") {
				return nil, errors.New("Ignore paths may not contain Windows-style path separators: " + val)
			}

			text := strings.ReplaceAll(val, "*", ".*")
			re, err := regexp.Compile(text)
			if err != nil {
				LogSevere("Unable to compile regex: " + text)
				return nil, err
			}

			result.pathExcludePatterns = append(result.pathExcludePatterns, re)

		}
	}

	return &result, nil
}

func (p *PathFilter) IsFilteredOutByFilename(pathParam string) bool {

	if strings.Contains(pathParam, "\\") {
		LogSevere("Parameter cannot contain Window-style file paths")
		return false
	}

	strArr := strings.Split(pathParam, "/")

	// filename := path.Base(file)
	// if filename == "." {
	// 	return false
	// }

	for _, filename := range strArr {

		for _, val := range p.filenameExcludePatterns {
			if val.MatchString(filename) {
				return true
			}
		}

	}

	return false
}

func (p *PathFilter) IsFilteredOutByPath(path string) bool {

	if strings.Contains(path, "\\") {
		LogSevere("Parameter cannot contain Window-style file paths")
		return false
	}

	for _, val := range p.pathExcludePatterns {
		if val.MatchString(path) {
			return true
		}
	}

	return false

}

func ConvertAbsolutePathWithUnixSeparatorsToProjectRelativePath(path string, rootPath string) *string {

	// Strip project parent directory from path:
	// If pathToMonitor is: /home/user/codewind/project
	// and watchEventPath is: /home/user/codewind/project/some-file.txt
	// then this will convert watchEventPath to /some-file.txt

	if strings.Contains(path, "\\") {
		LogSevere("Parameter cannot contain Window-style file paths")
		return nil
	}

	rootPath = StripTrailingForwardSlash(rootPath)

	if !strings.HasPrefix(path, rootPath) {
		// This shouldn't happen, and is thus severe
		LogSevere("Watch event '" + path + "' does not match project path '" + rootPath + "'")
		return nil
	}

	path = strings.ReplaceAll(path, rootPath, "")

	if len(path) == 0 {
		// Ignore the empty case
		return nil
	}

	return &path

}
