package main

import (
	"errors"
	"strings"
)

// splits the given path in a slice of path elements
// all / are removed
func splitPath(path string) []string {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	if strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}
	if len(path) == 0 {
		return []string{}
	}
	return strings.Split(path, "/")
}

// returns the realative path starting by removing basePath from newPath
func getRelativePath(basePath string, newPath string) ([]string, error) {
	baseComps := splitPath(basePath)
	newComps := splitPath(newPath)
	if len(baseComps) > len(newComps) {
		err := errors.New("Base Path is longer than requested path!")
		return nil, err
	}
	for i, _ := range baseComps {
		if strings.Compare(newComps[i], baseComps[i]) != 0 {
			err := errors.New("Base Path is not a prefix of newPath")
			return nil, err
		}
	}
	return newComps[len(baseComps):], nil
}

// checks if the given name is a command
// currently only knows about _hooks
func isCommand(name string) bool {
	for _, cmd := range []string{"_hooks"} {
		if strings.Compare(name, cmd) == 0 {
			return true
		}
	}
	return false
}

// checks if a command appers somewhere in the path
func containsCommand(components []string) bool {
	for _, comp := range components {
		if isCommand(comp) {
			return true
		}
	}
	return false
}
