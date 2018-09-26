package dataprovider

import (
	"strings"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

type pathFilter struct {
	mask []string
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func createPathFilter(path string) (filter *pathFilter, err error) {
	return &pathFilter{mask: strings.Split(path, ".")}, nil
}

func (filter *pathFilter) match(path string) (result bool) {
	maskIndex, pathIndex := 0, 0
	pathSlice := strings.Split(path, ".")
	maskSlice := filter.mask

	for maskIndex < len(maskSlice) && pathIndex < len(pathSlice) {
		if pathSlice[pathIndex] != maskSlice[maskIndex] && maskSlice[maskIndex] != "*" {
			return false
		}

		if maskSlice[maskIndex] == "*" {
			if pathIndex < len(pathSlice)-1 &&
				maskIndex < len(maskSlice)-1 &&
				pathSlice[pathIndex+1] != maskSlice[maskIndex+1] &&
				maskSlice[maskIndex+1] != "*" {
				pathIndex++
				continue
			}
		}

		pathIndex++
		maskIndex++
	}

	if maskIndex != len(maskSlice) {
		return false
	}

	return true
}
