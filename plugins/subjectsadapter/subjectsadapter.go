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

package subjectsadapter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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

type subjectsAdapter struct {
	subjects         []string
	subscribed       bool
	subscribeChannel chan map[string]interface{}
	config           adapterConfig
}

type adapterConfig struct {
	VISPath  string `json:"visPath"`
	FilePath string `json:"filePath"`
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// New creates adapter instance.
func New(configJSON json.RawMessage) (adapter dataprovider.DataAdapter, err error) {
	log.Info("Create Subjects adapter")

	localAdapter := &subjectsAdapter{subscribeChannel: make(chan map[string]interface{}, subscribeChannelSize)}

	if configJSON == nil {
		return nil, aoserrors.New("config should be set")
	}

	if err = json.Unmarshal(configJSON, &localAdapter.config); err != nil {
		return nil, aoserrors.Wrap(err)
	}

	if err = localAdapter.readSubjects(); err != nil {
		log.Warnf("Can't read subjects: %s. Empty subjects will be used", err)

		localAdapter.subjects = make([]string, 0)

		if err = os.MkdirAll(filepath.Dir(localAdapter.config.FilePath), 0o755); err != nil {
			return nil, aoserrors.Wrap(err)
		}
	}

	log.WithField("subjects", localAdapter.subjects).Debug("Subjects adapter")

	return localAdapter, nil
}

// Close closes adapter.
func (adapter *subjectsAdapter) Close() {
	log.Info("Close subjects adapter")
}

// GetName returns adapter name.
func (adapter *subjectsAdapter) GetName() (name string) {
	return "subjectsadapter"
}

// GetPathList returns list of all pathes for this adapter.
func (adapter *subjectsAdapter) GetPathList() (pathList []string, err error) {
	return []string{adapter.config.VISPath}, nil
}

// IsPathPublic returns true if requested data accessible without authorization.
func (adapter *subjectsAdapter) IsPathPublic(path string) (result bool, err error) {
	return true, nil
}

// GetData returns data by path.
func (adapter *subjectsAdapter) GetData(pathList []string) (data map[string]interface{}, err error) {
	log.WithField("subjects", adapter.subjects).Debug("Get subjects")

	data = make(map[string]interface{})

	for _, path := range pathList {
		if path == adapter.config.VISPath {
			data[path] = adapter.subjects
		} else {
			return nil, aoserrors.Errorf("path %s doesn't exits", path)
		}
	}

	return data, nil
}

// SetData sets data by pathes.
func (adapter *subjectsAdapter) SetData(data map[string]interface{}) (err error) {
	for path, value := range data {
		if path == adapter.config.VISPath {
			subjects, ok := value.([]interface{})
			if !ok {
				return aoserrors.Errorf("wrong value type for path %s", path)
			}

			adapter.subjects = []string{}

			for _, subject := range subjects {
				subjectStr, ok := subject.(string)
				if !ok {
					return aoserrors.Errorf("wrong element type for path %s", path)
				}

				adapter.subjects = append(adapter.subjects, subjectStr)
			}

			log.WithField("subjects", adapter.subjects).Debug("Set subjects")

			if adapter.subscribed {
				adapter.subscribeChannel <- map[string]interface{}{path: subjects}
			}

			if err = adapter.writeSubjects(); err != nil {
				return aoserrors.Wrap(err)
			}
		} else {
			return aoserrors.Errorf("path %s doesn't exits", path)
		}
	}

	return nil
}

// GetSubscribeChannel returns channel on which data changes will be sent.
func (adapter *subjectsAdapter) GetSubscribeChannel() (channel <-chan map[string]interface{}) {
	return adapter.subscribeChannel
}

// Subscribe subscribes for data changes.
func (adapter *subjectsAdapter) Subscribe(pathList []string) (err error) {
	for _, path := range pathList {
		if path == adapter.config.VISPath {
			adapter.subscribed = true
		} else {
			return aoserrors.Errorf("path %s doesn't exits", path)
		}
	}

	return nil
}

// Unsubscribe unsubscribes from data changes.
func (adapter *subjectsAdapter) Unsubscribe(pathList []string) (err error) {
	for _, path := range pathList {
		if path == adapter.config.VISPath {
			adapter.subscribed = false
		} else {
			return aoserrors.Errorf("path %s doesn't exits", path)
		}
	}

	return nil
}

// UnsubscribeAll unsubscribes from all data changes.
func (adapter *subjectsAdapter) UnsubscribeAll() (err error) {
	adapter.subscribed = false

	return nil
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func (adapter *subjectsAdapter) readSubjects() (err error) {
	file, err := os.Open(adapter.config.FilePath)
	if err != nil {
		return aoserrors.Wrap(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	adapter.subjects = nil

	for scanner.Scan() {
		adapter.subjects = append(adapter.subjects, scanner.Text())
	}

	return nil
}

func (adapter *subjectsAdapter) writeSubjects() (err error) {
	file, err := os.Create(adapter.config.FilePath)
	if err != nil {
		return aoserrors.Wrap(err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	for _, claim := range adapter.subjects {
		fmt.Fprintln(writer, claim)
	}

	return aoserrors.Wrap(writer.Flush())
}
