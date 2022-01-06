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

package telemetryemulatoradapter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/aoscloud/aos_vis/dataprovider"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

const (
	defaultUpdatePeriod = 500
)

// TelemetryEmulatorAdapter sensor emulator adapter
type TelemetryEmulatorAdapter struct {
	sensorURL   *url.URL
	cfg         config
	baseAdapter *dataprovider.BaseAdapter
}

type config struct {
	SensorURL     string
	UpdatePeriod  uint64
	PathPrefix    string
	PathConverter map[string]string
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// New creates adapter instance
func New(configJSON json.RawMessage) (adapter dataprovider.DataAdapter, err error) {
	log.Info("Create telemetry emulator adapter")

	cfg := config{UpdatePeriod: defaultUpdatePeriod, PathPrefix: "Signal.Emulator"}

	// Parse config
	err = json.Unmarshal(configJSON, &cfg)
	if err != nil {
		return nil, err
	}

	if cfg.SensorURL == "" {
		return nil, errors.New("sensor URL should be defined")
	}

	localAdapter := &TelemetryEmulatorAdapter{cfg: cfg}

	if localAdapter.sensorURL, err = url.Parse(localAdapter.cfg.SensorURL); err != nil {
		return nil, err
	}

	if localAdapter.baseAdapter, err = dataprovider.NewBaseAdapter(); err != nil {
		return nil, err
	}

	localAdapter.baseAdapter.Name = "TelemetryEmulatorAdapter"

	// Create data map
	data, err := localAdapter.getDataFromTelemetryEmulator()
	if err != nil {
		return nil, err
	}

	for path, value := range data {
		localAdapter.baseAdapter.Data[path] = &dataprovider.BaseData{Value: value}
	}

	// Create attributes
	localAdapter.baseAdapter.Data["Attribute.Emulator.rectangle_long0"] = &dataprovider.BaseData{}
	localAdapter.baseAdapter.Data["Attribute.Emulator.rectangle_lat0"] = &dataprovider.BaseData{}
	localAdapter.baseAdapter.Data["Attribute.Emulator.rectangle_long1"] = &dataprovider.BaseData{}
	localAdapter.baseAdapter.Data["Attribute.Emulator.rectangle_lat1"] = &dataprovider.BaseData{}
	localAdapter.baseAdapter.Data["Attribute.Emulator.to_rectangle"] = &dataprovider.BaseData{}
	localAdapter.baseAdapter.Data["Attribute.Emulator.stop"] = &dataprovider.BaseData{}
	localAdapter.baseAdapter.Data["Attribute.Emulator.tire_break"] = &dataprovider.BaseData{}

	go localAdapter.processData()

	return localAdapter, nil
}

// Close closes adapter
func (adapter *TelemetryEmulatorAdapter) Close() {
	log.Info("Close telemetry emulator adapter")

	adapter.baseAdapter.Close()
}

// GetName returns adapter name
func (adapter *TelemetryEmulatorAdapter) GetName() (name string) {
	return adapter.baseAdapter.GetName()
}

// GetPathList returns list of all pathes for this adapter
func (adapter *TelemetryEmulatorAdapter) GetPathList() (pathList []string, err error) {
	return adapter.baseAdapter.GetPathList()
}

// IsPathPublic returns true if requested data accessible without authorization
func (adapter *TelemetryEmulatorAdapter) IsPathPublic(path string) (result bool, err error) {
	adapter.baseAdapter.Mutex.Lock()
	defer adapter.baseAdapter.Mutex.Unlock()

	// TODO: return false, once authorization is integrated

	return true, nil
}

// GetData returns data by path
func (adapter *TelemetryEmulatorAdapter) GetData(pathList []string) (data map[string]interface{}, err error) {
	return adapter.baseAdapter.GetData(pathList)
}

// SetData sets data by pathes
func (adapter *TelemetryEmulatorAdapter) SetData(data map[string]interface{}) (err error) {
	sendData, err := convertVisFormatToData(data)
	if err != nil {
		return err
	}

	path, err := url.Parse("attributes/")
	if err != nil {
		return err
	}

	address := adapter.sensorURL.ResolveReference(path).String()

	log.WithField("url", address).Debugf("Set data to sensor emulator: %s", string(sendData))

	res, err := http.Post(address, "application/json", bytes.NewReader(sendData))
	if err != nil {
		return err
	}

	if res.StatusCode != 201 {
		return errors.New(res.Status)
	}

	return adapter.baseAdapter.SetData(data)
}

// GetSubscribeChannel returns channel on which data changes will be sent
func (adapter *TelemetryEmulatorAdapter) GetSubscribeChannel() (channel <-chan map[string]interface{}) {
	return adapter.baseAdapter.SubscribeChannel
}

// Subscribe subscribes for data changes
func (adapter *TelemetryEmulatorAdapter) Subscribe(pathList []string) (err error) {
	return adapter.baseAdapter.Subscribe(pathList)
}

// Unsubscribe unsubscribes from data changes
func (adapter *TelemetryEmulatorAdapter) Unsubscribe(pathList []string) (err error) {
	return adapter.baseAdapter.Unsubscribe(pathList)
}

// UnsubscribeAll unsubscribes from all data changes
func (adapter *TelemetryEmulatorAdapter) UnsubscribeAll() (err error) {
	return adapter.baseAdapter.UnsubscribeAll()
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func (adapter *TelemetryEmulatorAdapter) convertPath(inPath string) (outPath string) {
	var ok bool

	if outPath, ok = adapter.cfg.PathConverter[inPath]; !ok {
		return inPath
	}

	return outPath
}

func (adapter *TelemetryEmulatorAdapter) parseNode(prefix string, element interface{}) (visData map[string]interface{}) {
	visData = make(map[string]interface{})

	m, ok := element.(map[string]interface{})
	if ok {
		for path, value := range m {
			if prefix != "" {
				path = prefix + "." + path
			}

			for visPath, visValue := range adapter.parseNode(path, value) {
				visData[adapter.convertPath(visPath)] = visValue
			}
		}
	} else {
		visData[adapter.convertPath(prefix)] = element
	}

	return visData
}

func (adapter *TelemetryEmulatorAdapter) convertDataToVisFormat(dataJSON []byte) (visData map[string]interface{}, err error) {
	var data interface{}

	err = json.Unmarshal(dataJSON, &data)
	if err != nil {
		return visData, err
	}

	visData = adapter.parseNode(adapter.cfg.PathPrefix, data)

	return visData, nil
}

func (adapter *TelemetryEmulatorAdapter) getDataFromTelemetryEmulator() (visData map[string]interface{}, err error) {
	path, err := url.Parse("stats")
	if err != nil {
		return visData, err
	}

	address := adapter.sensorURL.ResolveReference(path).String()

	res, err := http.Get(address)
	if err != nil {
		return visData, err
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return visData, err
	}

	res.Body.Close()

	log.WithField("url", address).Debugf("Get data from sensor emulator: %s", string(data))

	return adapter.convertDataToVisFormat(data)
}

func (adapter *TelemetryEmulatorAdapter) processData() {
	ticker := time.NewTicker(time.Duration(adapter.cfg.UpdatePeriod) * time.Millisecond)

	for {
		<-ticker.C

		data, err := adapter.getDataFromTelemetryEmulator()
		if err != nil {
			log.Errorf("Can't read data: %s", err)
			continue
		}

		if err = adapter.baseAdapter.SetData(data); err != nil {
			log.Errorf("Can't update data: %s", err)
			continue
		}
	}
}

func convertVisFormatToData(visData map[string]interface{}) (dataJSON []byte, err error) {
	sendData := make(map[string]interface{})

	for path, value := range visData {
		if strings.HasPrefix(path, "Attribute.Emulator.") {
			path = strings.TrimPrefix(path, "Attribute.Emulator.")
			sendData[path] = value
		} else {
			return dataJSON, fmt.Errorf("path %s does not exist", path)
		}
	}

	dataJSON, err = json.Marshal(&sendData)
	if err != nil {
		return dataJSON, err
	}

	return dataJSON, nil
}
