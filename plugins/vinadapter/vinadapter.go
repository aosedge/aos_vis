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

package vinadapter

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/aoscloud/aos_common/aoserrors"
	"github.com/aoscloud/aos_vis/dataprovider"
)

/*******************************************************************************
 * Consts
 ******************************************************************************/

const vinLength = 17

/*******************************************************************************
 * Types
 ******************************************************************************/

type vinAdapter struct {
	vin    string
	config adapterConfig
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
	log.Info("Create VIN adapter")

	localAdapter := new(vinAdapter)

	if configJSON == nil {
		return nil, aoserrors.New("config should be set")
	}

	if err = json.Unmarshal(configJSON, &localAdapter.config); err != nil {
		return nil, aoserrors.Wrap(err)
	}

	vin, err := ioutil.ReadFile(localAdapter.config.FilePath)
	if err != nil {
		vin = generateVIN()

		log.Warnf("Can't read VIN: %s. Generate new one: %s", err, string(vin))

		if err = ioutil.WriteFile(localAdapter.config.FilePath, vin, 0o600); err != nil {
			return nil, aoserrors.Wrap(err)
		}
	}

	localAdapter.vin = string(vin)

	log.WithField("VIN", localAdapter.vin).Debug("VIN adapter")

	return localAdapter, nil
}

// Close closes adapter.
func (adapter *vinAdapter) Close() {
	log.Info("Close VIN adapter")
}

// GetName returns adapter name.
func (adapter *vinAdapter) GetName() (name string) {
	return "vinadapter"
}

// GetPathList returns list of all pathes for this adapter.
func (adapter *vinAdapter) GetPathList() (pathList []string, err error) {
	return []string{adapter.config.VISPath}, nil
}

// IsPathPublic returns true if requested data accessible without authorization.
func (adapter *vinAdapter) IsPathPublic(path string) (result bool, err error) {
	return true, nil
}

// GetData returns data by path.
func (adapter *vinAdapter) GetData(pathList []string) (data map[string]interface{}, err error) {
	data = make(map[string]interface{})

	for _, path := range pathList {
		if path == adapter.config.VISPath {
			data[path] = adapter.vin
		} else {
			return nil, aoserrors.Errorf("path %s doesn't exits", path)
		}
	}

	return data, nil
}

// SetData sets data by pathes.
func (adapter *vinAdapter) SetData(data map[string]interface{}) (err error) {
	if len(data) == 0 {
		return nil
	}

	for _, path := range data {
		if path != adapter.config.VISPath {
			return aoserrors.Errorf("path %s doesn't exits", path)
		}
	}

	return aoserrors.Errorf("signal %s cannot be set since it is a read only signal", adapter.config.VISPath)
}

// GetSubscribeChannel returns channel on which data changes will be sent.
func (adapter *vinAdapter) GetSubscribeChannel() (channel <-chan map[string]interface{}) {
	return nil
}

// Subscribe subscribes for data changes.
func (adapter *vinAdapter) Subscribe(pathList []string) (err error) {
	return nil
}

// Unsubscribe unsubscribes from data changes.
func (adapter *vinAdapter) Unsubscribe(pathList []string) (err error) {
	return nil
}

// UnsubscribeAll unsubscribes from all data changes.
func (adapter *vinAdapter) UnsubscribeAll() (err error) {
	return nil
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func generateVIN() (vin []byte) {
	const vinSymbols = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	vin = make([]byte, vinLength)

	rand.Seed(time.Now().UnixNano())

	for i := range vin {
		vin[i] = vinSymbols[rand.Intn(len(vinSymbols))] // nolint:gosec // test implementation
	}

	return vin
}
