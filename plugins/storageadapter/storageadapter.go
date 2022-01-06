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

package storageadapter

import (
	"bytes"
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/aoscloud/aos_vis/dataprovider"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// StorageAdapter storage adapter
type StorageAdapter struct {
	baseAdapter *dataprovider.BaseAdapter
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// New creates adapter instance
func New(configJSON json.RawMessage) (adapter dataprovider.DataAdapter, err error) {
	log.Info("Create storage adapter")

	localAdapter := new(StorageAdapter)

	localAdapter.baseAdapter, err = dataprovider.NewBaseAdapter()
	if err != nil {
		return nil, err
	}

	localAdapter.baseAdapter.Name = "StorageAdapter"

	var sensors struct {
		Data map[string]*dataprovider.BaseData
	}

	// Parse config
	decoder := json.NewDecoder(bytes.NewReader(configJSON))

	decoder.UseNumber()

	if err = decoder.Decode(&sensors); err != nil {
		return nil, err
	}

	localAdapter.baseAdapter.Data = sensors.Data

	return localAdapter, nil
}

// Close closes adapter
func (adapter *StorageAdapter) Close() {
	log.Info("Close storage adapter")

	adapter.baseAdapter.Close()
}

// GetName returns adapter name
func (adapter *StorageAdapter) GetName() (name string) {
	return adapter.baseAdapter.GetName()
}

// GetPathList returns list of all pathes for this adapter
func (adapter *StorageAdapter) GetPathList() (pathList []string, err error) {
	return adapter.baseAdapter.GetPathList()
}

// IsPathPublic returns true if requested data accessible without authorization
func (adapter *StorageAdapter) IsPathPublic(path string) (result bool, err error) {
	return adapter.baseAdapter.IsPathPublic(path)
}

// GetData returns data by path
func (adapter *StorageAdapter) GetData(pathList []string) (data map[string]interface{}, err error) {
	return adapter.baseAdapter.GetData(pathList)
}

// SetData sets data by pathes
func (adapter *StorageAdapter) SetData(data map[string]interface{}) (err error) {
	return adapter.baseAdapter.SetData(data)
}

// GetSubscribeChannel returns channel on which data changes will be sent
func (adapter *StorageAdapter) GetSubscribeChannel() (channel <-chan map[string]interface{}) {
	return adapter.baseAdapter.SubscribeChannel
}

// Subscribe subscribes for data changes
func (adapter *StorageAdapter) Subscribe(pathList []string) (err error) {
	return adapter.baseAdapter.Subscribe(pathList)
}

// Unsubscribe unsubscribes from data changes
func (adapter *StorageAdapter) Unsubscribe(pathList []string) (err error) {
	return adapter.baseAdapter.Unsubscribe(pathList)
}

// UnsubscribeAll unsubscribes from all data changes
func (adapter *StorageAdapter) UnsubscribeAll() (err error) {
	return adapter.baseAdapter.UnsubscribeAll()
}
