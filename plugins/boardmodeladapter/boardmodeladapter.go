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

package boardmodeladapter

import (
	"encoding/json"
	"io/ioutil"

	log "github.com/sirupsen/logrus"

	"github.com/aoscloud/aos_common/aoserrors"
	"github.com/aoscloud/aos_vis/dataprovider"
)

/*******************************************************************************
 * Consts
 ******************************************************************************/

const defaultBoardModel = "board;1.0"

/*******************************************************************************
 * Types
 ******************************************************************************/

type boardModelAdapter struct {
	boardModel string
	config     adapterConfig
}

type adapterConfig struct {
	VISPath  string `json:"VISPath"`
	FilePath string `json:"FilePath"`
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// New creates adapter instance.
func New(configJSON json.RawMessage) (adapter dataprovider.DataAdapter, err error) {
	log.Info("Create BoardModel adapter")

	localAdapter := new(boardModelAdapter)

	if configJSON == nil {
		return nil, aoserrors.New("config should be set")
	}

	if err = json.Unmarshal(configJSON, &localAdapter.config); err != nil {
		return nil, aoserrors.Wrap(err)
	}

	boardModel, err := ioutil.ReadFile(localAdapter.config.FilePath)
	if err != nil {
		log.Warnf("Can't read board model: %s. Use default one: %s", err, defaultBoardModel)

		if err = ioutil.WriteFile(localAdapter.config.FilePath, []byte(defaultBoardModel), 0o644); err != nil {
			return nil, aoserrors.Wrap(err)
		}

		boardModel = []byte(defaultBoardModel)
	}

	localAdapter.boardModel = string(boardModel)

	log.WithField("Board model", localAdapter.boardModel).Debug("BoardModel adapter")

	return localAdapter, nil
}

// Close closes adapter.
func (adapter *boardModelAdapter) Close() {
	log.Info("Close BoardModel adapter")
}

// GetName returns adapter name.
func (adapter *boardModelAdapter) GetName() (name string) {
	return "boardmodeladapter"
}

// GetPathList returns list of all pathes for this adapter.
func (adapter *boardModelAdapter) GetPathList() (pathList []string, err error) {
	return []string{adapter.config.VISPath}, nil
}

// IsPathPublic returns true if requested data accessible without authorization.
func (adapter *boardModelAdapter) IsPathPublic(path string) (result bool, err error) {
	return true, nil
}

// GetData returns data by path.
func (adapter *boardModelAdapter) GetData(pathList []string) (data map[string]interface{}, err error) {
	data = make(map[string]interface{})

	for _, path := range pathList {
		if path == adapter.config.VISPath {
			data[path] = adapter.boardModel
		} else {
			return nil, aoserrors.Errorf("path %s doesn't exits", path)
		}
	}

	return data, nil
}

// SetData sets data by pathes.
func (adapter *boardModelAdapter) SetData(data map[string]interface{}) (err error) {
	if len(data) == 0 {
		return nil
	}

	for _, path := range data {
		if path != adapter.config.VISPath {
			return aoserrors.Errorf("path %s doesn't exits", path)
		}
	}

	return aoserrors.Errorf("signal %s cannot be set since it is a read only attribute", adapter.config.VISPath)
}

// GetSubscribeChannel returns channel on which data changes will be sent.
func (adapter *boardModelAdapter) GetSubscribeChannel() (channel <-chan map[string]interface{}) {
	return nil
}

// Subscribe subscribes for data changes.
func (adapter *boardModelAdapter) Subscribe(pathList []string) (err error) {
	return nil
}

// Unsubscribe unsubscribes from data changes.
func (adapter *boardModelAdapter) Unsubscribe(pathList []string) (err error) {
	return nil
}

// UnsubscribeAll unsubscribes from all data changes.
func (adapter *boardModelAdapter) UnsubscribeAll() (err error) {
	return nil
}
