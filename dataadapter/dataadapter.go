// SPDX-License-Identifier: Apache-2.0
//
// Copyright 2019 Renesas Inc.
// Copyright 2019 EPAM Systems Inc.
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

package dataadapter

import (
	"errors"
	"plugin"
)

// DataAdapter interface for working with real data
type DataAdapter interface {
	// Close closes adapter
	Close()
	// GetName returns adapter name
	GetName() (name string)
	// GetPathList returns list of all pathes for this adapter
	GetPathList() (pathList []string, err error)
	// IsPathPublic returns true if requested data accessible without authorization
	IsPathPublic(path string) (result bool, err error)
	// GetData returns data by path
	GetData(pathList []string) (data map[string]interface{}, err error)
	// SetData sets data by pathes
	SetData(data map[string]interface{}) (err error)
	// GetSubscribeChannel returns channel on which data changes will be sent
	GetSubscribeChannel() (channel <-chan map[string]interface{})
	// Subscribe subscribes for data changes
	Subscribe(pathList []string) (err error)
	// Unsubscribe unsubscribes from data changes
	Unsubscribe(pathList []string) (err error)
	// UnsubscribeAll unsubscribes from all data changes
	UnsubscribeAll() (err error)
}

// NewAdapter creates new adapter instance
func NewAdapter(pluginPath string, configJSON []byte) (adapter DataAdapter, err error) {
	plugin, err := plugin.Open(pluginPath)
	if err != nil {
		return adapter, err
	}

	newAdapterSymbol, err := plugin.Lookup("NewAdapter")
	if err != nil {
		return adapter, err
	}

	newAdapterFunction, ok := newAdapterSymbol.(func(configJSON []byte) (DataAdapter, error))
	if !ok {
		return adapter, errors.New("unexpected function type")
	}

	return newAdapterFunction(configJSON)
}
