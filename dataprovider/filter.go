// SPDX-License-Identifier: Apache-2.0
//
// Copyright (C) 2021 Renesas Electronics Corporation.
// Copyright (C) 2021 EPAM Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dataprovider

import (
	"strings"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// PathFilter path filter structure.
type PathFilter struct {
	mask []string
}

/*******************************************************************************
 * Private
 ******************************************************************************/

// CreatePathFilter creates path filter.
func CreatePathFilter(path string) (filter *PathFilter, err error) {
	return &PathFilter{mask: strings.Split(path, ".")}, nil
}

// Match returns true is path matches the filter.
func (filter *PathFilter) Match(path string) (result bool) {
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

	return maskIndex == len(maskSlice)
}
