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

package unitmodeladapter

import (
	"encoding/json"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/aoscloud/aos_common/aoserrors"
	"github.com/aoscloud/aos_vis/dataprovider"
)

/*******************************************************************************
 * Consts
 ******************************************************************************/

const defaultUnitModel = "unit;1.0"

/*******************************************************************************
 * Types
 ******************************************************************************/

type unitModelAdapter struct {
	unitModel string
	config    adapterConfig
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
	log.Info("Create unit model adapter")

	localAdapter := new(unitModelAdapter)

	if configJSON == nil {
		return nil, aoserrors.New("config should be set")
	}

	if err = json.Unmarshal(configJSON, &localAdapter.config); err != nil {
		return nil, aoserrors.Wrap(err)
	}

	unitModel, err := os.ReadFile(localAdapter.config.FilePath)
	if err != nil {
		log.Warnf("Can't read unit model: %s. Use default one: %s", err, defaultUnitModel)

		if err = os.WriteFile(localAdapter.config.FilePath,
			[]byte(defaultUnitModel), 0o600); err != nil {
			return nil, aoserrors.Wrap(err)
		}

		unitModel = []byte(defaultUnitModel)
	}

	localAdapter.unitModel = string(unitModel)

	log.WithField("Unit model", localAdapter.unitModel).Debug("UnitModel adapter")

	return localAdapter, nil
}

// Close closes adapter.
func (adapter *unitModelAdapter) Close() {
	log.Info("Close UnitModel adapter")
}

// GetName returns adapter name.
func (adapter *unitModelAdapter) GetName() (name string) {
	return "unitmodeladapter"
}

// GetPathList returns list of all paths for this adapter.
func (adapter *unitModelAdapter) GetPathList() (pathList []string, err error) {
	return []string{adapter.config.VISPath}, nil
}

// IsPathPublic returns true if requested data accessible without authorization.
func (adapter *unitModelAdapter) IsPathPublic(path string) (result bool, err error) {
	return true, nil
}

// GetData returns data by path.
func (adapter *unitModelAdapter) GetData(pathList []string) (data map[string]interface{}, err error) {
	data = make(map[string]interface{})

	for _, path := range pathList {
		if path == adapter.config.VISPath {
			data[path] = adapter.unitModel
		} else {
			return nil, aoserrors.Errorf("path %s doesn't exits", path)
		}
	}

	return data, nil
}

// SetData sets data by paths.
func (adapter *unitModelAdapter) SetData(data map[string]interface{}) (err error) {
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
func (adapter *unitModelAdapter) GetSubscribeChannel() (channel <-chan map[string]interface{}) {
	return nil
}

// Subscribe subscribes for data changes.
func (adapter *unitModelAdapter) Subscribe(pathList []string) (err error) {
	return nil
}

// Unsubscribe unsubscribes from data changes.
func (adapter *unitModelAdapter) Unsubscribe(pathList []string) (err error) {
	return nil
}

// UnsubscribeAll unsubscribes from all data changes.
func (adapter *unitModelAdapter) UnsubscribeAll() (err error) {
	return nil
}
