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

package usersadapter

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/aoscloud/aos_common/aoserrors"
	"github.com/aoscloud/aos_vis/dataprovider"
)

/*******************************************************************************
 * Consts
 ******************************************************************************/

const subscribeChannelSize = 2

/*******************************************************************************
 * Types
 ******************************************************************************/

type usersAdapter struct {
	users            []string
	subscribed       bool
	subscribeChannel chan map[string]interface{}
	config           adapterConfig
}

type adapterConfig struct {
	VISPath  string `json:"VISPath"`
	FilePath string `json:"FilePath"`
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// New creates adapter instance
func New(configJSON json.RawMessage) (adapter dataprovider.DataAdapter, err error) {
	log.Info("Create Users adapter")

	localAdapter := &usersAdapter{subscribeChannel: make(chan map[string]interface{}, subscribeChannelSize)}

	if configJSON == nil {
		return nil, errors.New("config should be set")
	}

	if err = json.Unmarshal(configJSON, &localAdapter.config); err != nil {
		return nil, aoserrors.Wrap(err)
	}

	if err = localAdapter.readUsers(); err != nil {
		log.Warnf("Can't read users: %s. Empty users will be used", err)

		localAdapter.users = make([]string, 0)
	}

	log.WithField("users", localAdapter.users).Debug("Users adapter")

	return localAdapter, nil
}

// Close closes adapter
func (adapter *usersAdapter) Close() {
	log.Info("Close Users adapter")
}

// GetName returns adapter name
func (adapter *usersAdapter) GetName() (name string) {
	return "usersadapter"
}

// GetPathList returns list of all pathes for this adapter
func (adapter *usersAdapter) GetPathList() (pathList []string, err error) {
	return []string{adapter.config.VISPath}, nil
}

// IsPathPublic returns true if requested data accessible without authorization
func (adapter *usersAdapter) IsPathPublic(path string) (result bool, err error) {
	return true, nil
}

// GetData returns data by path
func (adapter *usersAdapter) GetData(pathList []string) (data map[string]interface{}, err error) {
	log.WithField("users", adapter.users).Debug("Get Users")

	data = make(map[string]interface{})

	for _, path := range pathList {
		if path == adapter.config.VISPath {
			data[path] = adapter.users
		} else {
			return nil, fmt.Errorf("path %s doesn't exits", path)
		}
	}

	return data, nil
}

// SetData sets data by pathes
func (adapter *usersAdapter) SetData(data map[string]interface{}) (err error) {
	for path, value := range data {
		if path == adapter.config.VISPath {
			users, ok := value.([]interface{})
			if !ok {
				return fmt.Errorf("wrong value type for path %s", path)
			}

			adapter.users = []string{}

			for _, user := range users {
				userStr, ok := user.(string)
				if !ok {
					return fmt.Errorf("wrong element type for path %s", path)
				}

				adapter.users = append(adapter.users, userStr)
			}

			log.WithField("users", adapter.users).Debug("Set Users")

			if adapter.subscribed {
				adapter.subscribeChannel <- map[string]interface{}{path: users}
			}

			if err = adapter.writeUsers(); err != nil {
				return aoserrors.Wrap(err)
			}
		} else {
			return fmt.Errorf("path %s doesn't exits", path)
		}
	}

	return nil
}

// GetSubscribeChannel returns channel on which data changes will be sent
func (adapter *usersAdapter) GetSubscribeChannel() (channel <-chan map[string]interface{}) {
	return adapter.subscribeChannel
}

// Subscribe subscribes for data changes
func (adapter *usersAdapter) Subscribe(pathList []string) (err error) {
	for _, path := range pathList {
		if path == adapter.config.VISPath {
			adapter.subscribed = true
		} else {
			return fmt.Errorf("path %s doesn't exits", path)
		}
	}

	return nil
}

// Unsubscribe unsubscribes from data changes
func (adapter *usersAdapter) Unsubscribe(pathList []string) (err error) {
	for _, path := range pathList {
		if path == adapter.config.VISPath {
			adapter.subscribed = false
		} else {
			return fmt.Errorf("path %s doesn't exits", path)
		}
	}

	return nil
}

// UnsubscribeAll unsubscribes from all data changes
func (adapter *usersAdapter) UnsubscribeAll() (err error) {
	adapter.subscribed = false

	return nil
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func (adapter *usersAdapter) readUsers() (err error) {
	file, err := os.Open(adapter.config.FilePath)
	if err != nil {
		return aoserrors.Wrap(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	adapter.users = nil

	for scanner.Scan() {
		adapter.users = append(adapter.users, scanner.Text())
	}

	return nil
}

func (adapter *usersAdapter) writeUsers() (err error) {
	file, err := os.Create(adapter.config.FilePath)
	if err != nil {
		return aoserrors.Wrap(err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	for _, claim := range adapter.users {
		fmt.Fprintln(writer, claim)
	}

	return aoserrors.Wrap(writer.Flush())
}
