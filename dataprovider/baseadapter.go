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
	"fmt"
	"reflect"
	"sync"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// BaseAdapter base adapter.
type BaseAdapter struct {
	Name string
	Data map[string]*BaseData
	sync.Mutex
	SubscribeChannel chan map[string]interface{}
}

// BaseData base data type.
type BaseData struct {
	Public    bool
	ReadOnly  bool
	Value     interface{}
	subscribe bool
}

/*******************************************************************************
 * Private
 ******************************************************************************/

// NewBaseAdapter creates base adapter.
func NewBaseAdapter() (adapter *BaseAdapter, err error) {
	adapter = new(BaseAdapter)

	adapter.Data = make(map[string]*BaseData)
	adapter.SubscribeChannel = make(chan map[string]interface{}, subscribeChannelSize)

	return adapter, nil
}

// Close closes adapter.
func (adapter *BaseAdapter) Close() {
}

// GetName returns adapter name.
func (adapter *BaseAdapter) GetName() (name string) {
	return adapter.Name
}

// GetPathList returns list of all pathes for this adapter.
func (adapter *BaseAdapter) GetPathList() (pathList []string, err error) {
	adapter.Lock()
	defer adapter.Unlock()

	pathList = make([]string, 0, len(adapter.Data))

	for path := range adapter.Data {
		pathList = append(pathList, path)
	}

	return pathList, nil
}

// IsPathPublic returns true if requested data accessible without authorization.
func (adapter *BaseAdapter) IsPathPublic(path string) (result bool, err error) {
	adapter.Lock()
	defer adapter.Unlock()

	if _, ok := adapter.Data[path]; !ok {
		return result, fmt.Errorf("path %s doesn't exits", path)
	}

	return adapter.Data[path].Public, nil
}

// GetData returns data by path.
func (adapter *BaseAdapter) GetData(pathList []string) (data map[string]interface{}, err error) {
	adapter.Lock()
	defer adapter.Unlock()

	data = make(map[string]interface{})

	for _, path := range pathList {
		if _, ok := adapter.Data[path]; !ok {
			return data, fmt.Errorf("path %s doesn't exits", path)
		}

		data[path] = adapter.Data[path].Value
	}

	return data, nil
}

// SetData sets data by pathes.
func (adapter *BaseAdapter) SetData(data map[string]interface{}) (err error) {
	adapter.Lock()
	defer adapter.Unlock()

	changedData := make(map[string]interface{})

	for path, value := range data {
		if _, ok := adapter.Data[path]; !ok {
			return fmt.Errorf("path %s doesn't exits", path)
		}

		if adapter.Data[path].ReadOnly {
			return fmt.Errorf("signal %s cannot be set since it is a read only signal", path)
		}

		oldValue := adapter.Data[path].Value
		adapter.Data[path].Value = value

		if !reflect.DeepEqual(oldValue, value) && adapter.Data[path].subscribe {
			changedData[path] = value
		}
	}

	if len(changedData) > 0 {
		adapter.SubscribeChannel <- changedData
	}

	return nil
}

// GetSubscribeChannel returns channel on which data changes will be sent.
func (adapter *BaseAdapter) GetSubscribeChannel() (channel <-chan map[string]interface{}) {
	return adapter.SubscribeChannel
}

// Subscribe subscribes for data changes.
func (adapter *BaseAdapter) Subscribe(pathList []string) (err error) {
	adapter.Lock()
	defer adapter.Unlock()

	for _, path := range pathList {
		if _, ok := adapter.Data[path]; !ok {
			return fmt.Errorf("path %s doesn't exits", path)
		}

		adapter.Data[path].subscribe = true
	}

	return nil
}

// Unsubscribe unsubscribes from data changes.
func (adapter *BaseAdapter) Unsubscribe(pathList []string) (err error) {
	adapter.Lock()
	defer adapter.Unlock()

	for _, path := range pathList {
		if _, ok := adapter.Data[path]; !ok {
			return fmt.Errorf("path %s doesn't exits", path)
		}

		adapter.Data[path].subscribe = false
	}

	return nil
}

// UnsubscribeAll unsubscribes from all data changes.
func (adapter *BaseAdapter) UnsubscribeAll() (err error) {
	adapter.Lock()
	defer adapter.Unlock()

	for _, data := range adapter.Data {
		data.subscribe = false
	}

	return nil
}
