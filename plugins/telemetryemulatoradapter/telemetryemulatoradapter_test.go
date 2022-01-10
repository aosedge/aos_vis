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

package telemetryemulatoradapter_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/aoscloud/aos_vis/dataadaptertest"
	"github.com/aoscloud/aos_vis/plugins/telemetryemulatoradapter"
)

/*******************************************************************************
 * Var
 ******************************************************************************/

var (
	adapterInfo  dataadaptertest.TestAdapterInfo
	emulatorData map[string]interface{}
)

/*******************************************************************************
 * Init
 ******************************************************************************/

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		TimestampFormat:  "2006-01-02 15:04:05.000",
		FullTimestamp:    true,
	})
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
}

/*******************************************************************************
 * Main
 ******************************************************************************/

func TestMain(m *testing.M) {
	startHTTPServer()

	telemetryEmulatorAdapter, err := telemetryemulatoradapter.New([]byte(`{"SensorURL":"http://localhost:8801"}`))
	if err != nil {
		log.Fatalf("Can't create sensor emulator adapter: %s", err)
	}
	defer telemetryEmulatorAdapter.Close()

	adapterInfo = dataadaptertest.TestAdapterInfo{
		Name:    "TelemetryEmulatorAdapter",
		Adapter: telemetryEmulatorAdapter,
		SetData: map[string]interface{}{
			"Attribute.Emulator.rectangle_long0": 23.56,
			"Attribute.Emulator.rectangle_lat0":  34.12,
			"Attribute.Emulator.rectangle_long1": 36.87,
			"Attribute.Emulator.rectangle_lat1":  39.21,
			"Attribute.Emulator.to_rectangle":    true,
		},
		SubscribeList: []string{
			"Attribute.Emulator.rectangle_long0",
			"Attribute.Emulator.rectangle_lat0",
			"Attribute.Emulator.rectangle_long1",
			"Attribute.Emulator.rectangle_lat1",
			"Attribute.Emulator.to_rectangle",
		},
		SetSubscribeData: map[string]interface{}{
			"Attribute.Emulator.rectangle_long0": 26.56,
			"Attribute.Emulator.rectangle_lat0":  38.12,
			"Attribute.Emulator.rectangle_long1": 40.87,
			"Attribute.Emulator.rectangle_lat1":  55.21,
			"Attribute.Emulator.to_rectangle":    false,
		},
	}

	ret := m.Run()

	defer os.Exit(ret)
}

/*******************************************************************************
 * Tests
 ******************************************************************************/

func TestGetName(t *testing.T) {
	if err := dataadaptertest.GetName(&adapterInfo); err != nil {
		t.Errorf("Test get name error: %s", err)
	}
}

func TestGetPathList(t *testing.T) {
	if err := dataadaptertest.GetPathList(&adapterInfo); err != nil {
		t.Errorf("Test get path lis error: %s", err)
	}
}

func TestPublicPath(t *testing.T) {
	if err := dataadaptertest.PublicPath(&adapterInfo); err != nil {
		t.Errorf("Test public path error: %s", err)
	}
}

func TestGetSetData(t *testing.T) {
	if err := dataadaptertest.GetSetData(&adapterInfo); err != nil {
		t.Errorf("Test get set data error: %s", err)
	}
}

func TestSubscribeUnsubscribe(t *testing.T) {
	if err := dataadaptertest.SubscribeUnsubscribe(&adapterInfo); err != nil {
		t.Errorf("Test subscribe unsubscribe error: %s", err)
	}
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func statsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	dataJSON, err := json.Marshal(emulatorData)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	if _, err := w.Write(dataJSON); err != nil {
		log.Errorf("Can't write response: %s", err)
	}
}

func attributesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	dataJSON, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error(err)

		w.WriteHeader(http.StatusBadRequest)

		if _, err := w.Write([]byte(err.Error())); err != nil {
			log.Errorf("Can't write response: %s", err)
		}

		return
	}

	r.Body.Close()

	if err = json.Unmarshal(dataJSON, &emulatorData); err != nil {
		log.Error(err)

		w.WriteHeader(http.StatusBadRequest)

		if _, err := w.Write([]byte(err.Error())); err != nil {
			log.Errorf("Can't write response: %s", err)
		}

		return
	}

	w.WriteHeader(http.StatusCreated)
}

func startHTTPServer() {
	emulatorData = map[string]interface{}{
		"rectangle_lat0":  nil,
		"rectangle_lat1":  nil,
		"rectangle_long0": nil,
		"rectangle_long1": nil,
		"to_rectangle":    nil,
	}

	http.HandleFunc("/stats/", statsHandler)
	http.HandleFunc("/attributes/", attributesHandler)

	go func() {
		if err := http.ListenAndServe("localhost:8801", nil); err != nil {
			log.Errorf("Can't serve http server: %s", err)
		}
	}()

	time.Sleep(1 * time.Second)
}
