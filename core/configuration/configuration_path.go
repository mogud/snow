package configuration

import (
	"strings"
)

const KeyDelimiter = ":"

func pathCombine(list []string) string {
	return strings.Join(list, KeyDelimiter)
}

func pathSectionKey(path string) string {
	idx := strings.LastIndexByte(path, ':')
	if idx == -1 {
		return path
	}

	return path[idx+1:]
}

func pathParent(path string) string {
	idx := strings.LastIndexByte(path, ':')
	if idx == -1 {
		return ""
	}

	return path[:idx]
}
