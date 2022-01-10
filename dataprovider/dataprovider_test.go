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

package dataprovider_test

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/aoscloud/aos_common/aoserrors"
	"github.com/aoscloud/aos_vis/config"
	"github.com/aoscloud/aos_vis/dataprovider"
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
 * Vars
 ******************************************************************************/

var provider *dataprovider.DataProvider

/*******************************************************************************
 * Main
 ******************************************************************************/

func TestMain(m *testing.M) {
	var err error

	configJSON := `{
	"Adapters":[
		{
			"Plugin":"testadapter",
			"Params": {
				"Data" : {
					"Attribute.Vehicle.VehicleIdentification.VIN":    {"Value": "TestVIN", "Public": true,"ReadOnly": true},
					"Attribute.Vehicle.UserIdentification.Users":     {"Value": ["User1", "Provider1"], "Public": true},

					"Signal.Drivetrain.InternalCombustionEngine.RPM": {"Value": 1000, "ReadOnly": true},

					"Signal.Body.Trunk.IsLocked":                     {"Value": false},
					"Signal.Body.Trunk.IsOpen":                       {"Value": true},

					"Signal.Cabin.Door.Row1.Right.IsLocked":          {"Value": true},
					"Signal.Cabin.Door.Row1.Right.Window.Position":   {"Value": 50},
					"Signal.Cabin.Door.Row1.Left.IsLocked":           {"Value": true},
					"Signal.Cabin.Door.Row1.Left.Window.Position":    {"Value": 23},
					"Signal.Cabin.Door.Row2.Right.IsLocked":          {"Value": false},
					"Signal.Cabin.Door.Row2.Right.Window.Position":   {"Value": 100},
					"Signal.Cabin.Door.Row2.Left.IsLocked":           {"Value": true},
					"Signal.Cabin.Door.Row2.Left.Window.Position":    {"Value": 0}
				}
			}
		}
	]
}`

	var cfg config.Config

	decoder := json.NewDecoder(strings.NewReader(configJSON))
	// Parse config
	if err = decoder.Decode(&cfg); err != nil {
		log.Fatalf("Can't parse config: %s", err)
	}

	dataprovider.RegisterPlugin("testadapter", func(configJSON json.RawMessage) (
		adapter dataprovider.DataAdapter, err error) {
		baseAdapter, err := dataprovider.NewBaseAdapter()
		if err != nil {
			return nil, aoserrors.Wrap(err)
		}

		var sensors struct {
			Data map[string]*dataprovider.BaseData
		}

		decoder := json.NewDecoder(bytes.NewReader(configJSON))
		decoder.UseNumber()
		if err = decoder.Decode(&sensors); err != nil {
			return nil, aoserrors.Wrap(err)
		}

		baseAdapter.Data = sensors.Data

		return baseAdapter, nil
	})

	provider, err = dataprovider.New(&cfg)
	if err != nil {
		log.Fatalf("Can't create data provider: %s", err)
	}

	ret := m.Run()

	os.Exit(ret)
}

/*******************************************************************************
 * Tests
 ******************************************************************************/

func TestGetData(t *testing.T) { // nolint:wsl
	/*
		client -> {
			"action": "get",
			"path": "Signal.Drivetrain.InternalCombustionEngine.RPM",
			"requestId": "8756"
		}

		receive <- {
			"action": "get",
			"requestId": "8756",
			"value": 2372,
			"timestamp": 1489985044000
		}
	*/

	data, err := provider.GetData("Signal.Drivetrain.InternalCombustionEngine.RPM", nil)
	if err != nil {
		t.Errorf("Can't get data: %s", err)
	} else if _, ok := data.(json.Number); !ok {
		t.Errorf("Wrong data type: %s", reflect.TypeOf(data))
	}

	/*
		client -> {
			"action": "get",
			"path": "Signal.Drivetrain.InternalCombustionEngine.*",
			"requestId": "8756"
		}

		receive <- {
			"action": "get",
			"requestId": "8756",
			"value": 2372,
			"timestamp": 1489985044000
		}
	*/

	data, err = provider.GetData("Signal.Drivetrain.InternalCombustionEngine.*", nil)
	if err != nil {
		t.Errorf("Can't get data: %s", err)
	} else {
		if value, ok := data.(map[string]interface{}); !ok {
			t.Errorf("Wrong data type: %s", reflect.TypeOf(data))
		} else if len(value) != 1 {
			t.Errorf("Wrong map size: %d", len(value))
		} else if _, ok = value["Signal.Drivetrain.InternalCombustionEngine.RPM"]; !ok {
			t.Error("Wrong map item")
		}
	}

	/*
		client -> {
			"action": "get",
			"path": "Signal.Body.Trunk",
			"requestId": "9078"
		}

		receive <- {
			"action": "get",
			"requestId": "9078",
			"value": { "Signal.Body.Trunk.IsLocked": false,
				"Signal.Body.Trunk.IsOpen": true },
			"timestamp": 1489985044000
		}
	*/

	data, err = provider.GetData("Signal.Body.Trunk", nil)
	if err != nil {
		t.Errorf("Can't get data: %s", err)
	} else if value, ok := data.(map[string]interface{}); !ok {
		t.Errorf("Wrong data type: %s", reflect.TypeOf(data))
	} else if len(value) != 2 {
		t.Errorf("Wrong map size: %d", len(value))
	}

	/*
		client -> {
			"action": "get",
			"path": "Signal.Cabin.Door.*.IsLocked",
			"requestId": "4523"
		}

		receive <- {
			"action": "get",
			"requestId": "4523",
			"value": [ {"Signal.Cabin.Door.Row1.Right.IsLocked" : true },
			           {"Signal.Cabin.Door.Row1.Left.IsLocked" : true },
				       {"Signal.Cabin.Door.Row2.Right.IsLocked" : false },
				       {"Signal.Cabin.Door.Row2.Left.IsLocked" : true } ],
			"timestamp": 1489985044000
		}
	*/

	data, err = provider.GetData("Signal.Cabin.Door.*.IsLocked", nil)
	if err != nil {
		t.Errorf("Can't get data: %s", err)
	} else if value, ok := data.([]map[string]interface{}); !ok {
		t.Errorf("Wrong data type: %s", reflect.TypeOf(data))
	} else {
		if len(value) != 4 {
			t.Errorf("Wrong array size: %d", len(value))
		}
		for _, item := range value {
			if len(item) != 1 {
				t.Errorf("Wrong map size: %d", len(item))
			}
		}
	}

	/*
		client -> {
			"action": "get",
			"path": "Signal.Cabin.Door.*",
			"requestId": "6745"
		}

		receive <- {
			"action": "get",
			"requestId": "6745",
			"value": [ {"Signal.Cabin.Door.Row1.Right.IsLocked" : true, "Signal.Cabin.Door.Row1.Right.Window.Position": 50},
			           {"Signal.Cabin.Door.Row1.Left.IsLocked" : true, "Signal.Cabin.Door.Row1.Left.Window.Position": 23},
			           {"Signal.Cabin.Door.Row2.Right.IsLocked" : false, "Signal.Cabin.Door.Row2.Right.Window.Position": 100 },
			           {"Signal.Cabin.Door.Row2.Left.IsLocked": true, "Signal.Cabin.Door.Row2.Left.Window.Position": 0 } ],
			"timestamp": 1489985044000
		}
	*/

	/*
		This case is unclear as Window.Position is subnode of Door.Row1.Right. We should define
		how to combine results in one map.
	*/

	data, err = provider.GetData("Signal.Cabin.Door.*", nil)
	if err != nil {
		t.Errorf("Can't get data: %s", err)
	} else if value, ok := data.([]map[string]interface{}); !ok {
		t.Errorf("Wrong data type: %s", reflect.TypeOf(data))
	} else {
		if len(value) != 8 {
			t.Errorf("Wrong array size: %d", len(value))
		}
		for _, item := range value {
			if len(item) != 1 {
				t.Errorf("Wrong map size: %d", len(item))
			}
		}
	}

	/*
		client -> {
			"action": "get",
			"path": "Body.Flux.Capacitor",
			"requestId": "1245"
		}

		receive <- {
			"action": "get",
			"requestId": "1245",
			"error": { "number":404,
				"reason": "invalid_path",
				"message": "The specified data path does not exist." },
			"timestamp": 1489985044000
		}
	*/

	if _, err = provider.GetData("Body.Flux.Capacitor", nil); err == nil {
		t.Error("Path should not exists")
	} else if !strings.Contains(err.Error(), "not exist") {
		t.Errorf("Wrong error type: %s", err)
	}
}

func TestSetData(t *testing.T) {
	if err := provider.SetData("Signal.Body.Trunk.IsLocked", true, nil); err != nil {
		t.Errorf("Can't set data: %s", err)
	}

	value, err := provider.GetData("Signal.Body.Trunk.IsLocked", nil)
	if err != nil {
		t.Errorf("Can't get data: %s", err)
	} else if value != true {
		t.Errorf("Data mistmatch: %v", value)
	}

	/*
		client -> {
			"action": "set",
			"path": "Signal.Cabin.Door.*.IsLocked",
			"value": [ {"Row1.Right.IsLocked": true },
			           {"Row1.Left.IsLocked": true },
			           {"Row2.Right.IsLocked": true },
			           {"Row2.Left.IsLocked": true } ],
			"requestId": "5689"
		}

		receive <- {
			"action": "set",
			"requestId": "5689",
			"timestamp": 1489985044000
		}
	*/

	if err = provider.SetData("Signal.Cabin.Door.*.IsLocked", []interface{}{
		map[string]interface{}{"Row1.Right.IsLocked": true},
		map[string]interface{}{"Row1.Left.IsLocked": true},
		map[string]interface{}{"Row2.Right.IsLocked": true},
		map[string]interface{}{"Row2.Left.IsLocked": true},
	}, nil); err != nil {
		t.Errorf("Can't set data: %s", err)
	} else if value, err = provider.GetData("Signal.Cabin.Door.*.IsLocked", nil); err != nil {
		t.Errorf("Can't get data: %s", err)
	} else {
		dataMap, err := arrayToMap(value)
		if err != nil {
			t.Error(err)
		}
		if dataMap["Signal.Cabin.Door.Row1.Right.IsLocked"] != true {
			t.Errorf("Data mistmatch: %v", value)
		}
		if dataMap["Signal.Cabin.Door.Row1.Left.IsLocked"] != true {
			t.Errorf("Data mistmatch: %v", value)
		}
		if dataMap["Signal.Cabin.Door.Row2.Right.IsLocked"] != true {
			t.Errorf("Data mistmatch: %v", value)
		}
		if dataMap["Signal.Cabin.Door.Row2.Left.IsLocked"] != true {
			t.Errorf("Data mistmatch: %v", value)
		}
	}

	/*
		client -> {
			"action": "set",
			"path": "Signal.Drivetrain.InternalCombustionEngine.RPM",
			"value": 2000,
			"requestId": "8912"
		}

		receive <- {
			"action": "set",
			"requestId": "8912",
			"error": { "number": 401,
			"reason": "read_only",
			"message": "The desired signal cannot be set since it is a read only signal"},
			"timestamp": 1489985044000
		}
	*/

	err = provider.SetData("Signal.Drivetrain.InternalCombustionEngine.RPM", 2000, nil)
	if err == nil {
		t.Error("Path should be read only")
	} else if !strings.Contains(err.Error(), "read only") {
		t.Errorf("Wrong error type: %s", err)
	}

	/*
		client -> {
			"action": "set",
			"path": "Signal.Drivetrain.InternalCombustionEngine.RPM",
			"value": { "locked" : true }
			"requestId": "2311"
			}

		receive <- {
			"action": "set",
			"requestId": "2311",
			"error": { "number": 400,
			"reason": "bad_request" ,
			"message": "The server is unable to fulfil the client request because the request is malformed."},
			"timestamp": 1489985044000
		}
	*/

	err = provider.SetData("Signal.Drivetrain.InternalCombustionEngine.RPM", map[string]interface{}{"locked": true}, nil)
	if err == nil {
		t.Error("Path should be read only")
	} else if !strings.Contains(err.Error(), "malformed") {
		t.Errorf("Wrong error type: %s", err)
	}
}

func TestPermissions(t *testing.T) {
	// Check public path for not authorized client
	_, err := provider.GetData("Attribute.Vehicle.VehicleIdentification.VIN", &dataprovider.AuthInfo{})
	if err != nil {
		t.Errorf("Can't get data: %s", err)
	}

	// Check private path for not authorized client
	_, err = provider.GetData("Signal.Drivetrain.InternalCombustionEngine.RPM", &dataprovider.AuthInfo{})
	if err == nil {
		t.Error("Path should not be accessible")
	} else if !strings.Contains(err.Error(), "not authorized") {
		t.Errorf("Wrong error type: %s", err)
	}

	// Check authorized but not permitted
	_, err = provider.GetData("Signal.Drivetrain.InternalCombustionEngine.RPM",
		&dataprovider.AuthInfo{IsAuthorized: true, Permissions: map[string]string{}})
	if err == nil {
		t.Error("Path should not be accessible")
	} else if !strings.Contains(err.Error(), "not have permissions") {
		t.Errorf("Wrong error type: %s", err)
	}

	// Check read permissions
	_, err = provider.GetData("Signal.Drivetrain.InternalCombustionEngine.RPM",
		&dataprovider.AuthInfo{
			IsAuthorized: true,
			Permissions:  map[string]string{"Signal.Drivetrain.InternalCombustionEngine.RPM": "r"},
		})
	if err != nil {
		t.Errorf("Can't get data: %s", err)
	}

	// Check no write permissions
	err = provider.SetData("Signal.Cabin.Door.Row1.Right.Window.Position", 0,
		&dataprovider.AuthInfo{IsAuthorized: true, Permissions: map[string]string{"Signal.Cabin.Door.*": "r"}})
	if err == nil {
		t.Error("Path should not be accessible")
	} else if !strings.Contains(err.Error(), "not have permissions") {
		t.Errorf("Wrong error type: %s", err)
	}

	// Check write permissions
	err = provider.SetData("Signal.Cabin.Door.Row1.Right.Window.Position", 0,
		&dataprovider.AuthInfo{IsAuthorized: true, Permissions: map[string]string{"Signal.Cabin.Door.*": "rw"}})
	if err != nil {
		t.Errorf("Can't set data: %s", err)
	}
}

func TestSubscribe(t *testing.T) {
	// Clear all locks
	if err := provider.SetData("Signal.Cabin.Door.*.IsLocked", []interface{}{
		map[string]interface{}{"Row1.Right.IsLocked": false},
		map[string]interface{}{"Row1.Left.IsLocked": false},
		map[string]interface{}{"Row2.Right.IsLocked": false},
		map[string]interface{}{"Row2.Left.IsLocked": false},
	}, nil); err != nil {
		t.Errorf("Can't set data: %s", err)
	}

	// Subscribes for all door locks
	id1, channel1, err := provider.Subscribe("Signal.Cabin.Door.*", nil)
	if err != nil {
		t.Errorf("Can't subscribe: %s", err)
	}

	// Subscribes for row1 door locks
	id2, channel2, err := provider.Subscribe("Signal.Cabin.Door.Row1.*", nil)
	if err != nil {
		t.Errorf("Can't get data: %s", err)
	}

	if len(provider.GetSubscribeIDs()) != 2 {
		t.Errorf("Wrong subscribers count: %d", len(provider.GetSubscribeIDs()))
	}

	// Set all locks
	if err = provider.SetData("Signal.Cabin.Door.*.IsLocked", []interface{}{
		map[string]interface{}{"Row1.Right.IsLocked": true},
		map[string]interface{}{"Row1.Left.IsLocked": true},
		map[string]interface{}{"Row2.Right.IsLocked": true},
		map[string]interface{}{"Row2.Left.IsLocked": true},
	}, nil); err != nil {
		t.Errorf("Can't set data: %s", err)
	}

	timeout := false
	eventChannel1 := false
	eventChannel2 := false

	for {
		select {
		case data := <-channel1:
			data1, err := arrayToMap(data)
			if err != nil {
				t.Error(err)
			}

			if len(data1) != 4 {
				t.Errorf("Wrong data size: %d", len(data1))
			}

			if data1["Signal.Cabin.Door.Row1.Right.IsLocked"] != true {
				t.Errorf("Data mistmatch: %v", false)
			}

			if data1["Signal.Cabin.Door.Row1.Left.IsLocked"] != true {
				t.Errorf("Data mistmatch: %v", false)
			}

			if data1["Signal.Cabin.Door.Row2.Right.IsLocked"] != true {
				t.Errorf("Data mistmatch: %v", false)
			}

			if data1["Signal.Cabin.Door.Row2.Left.IsLocked"] != true {
				t.Errorf("Data mistmatch: %v", false)
			}

			eventChannel1 = true
		case data := <-channel2:
			data2, err := arrayToMap(data)
			if err != nil {
				t.Error(err)
			}

			if len(data2) != 2 {
				t.Errorf("Wrong data size: %d", len(data2))
			}

			if data2["Signal.Cabin.Door.Row1.Right.IsLocked"] != true {
				t.Errorf("Data mistmatch: %v", false)
			}

			if data2["Signal.Cabin.Door.Row1.Left.IsLocked"] != true {
				t.Errorf("Data mistmatch: %v", false)
			}

			eventChannel2 = true
		case <-time.After(100 * time.Millisecond):
			timeout = true
		}

		if eventChannel1 && eventChannel2 {
			break
		}

		if timeout {
			t.Error("Waiting for data change timeout")
			break
		}
	}

	// Unsubscribes id1
	if err = provider.Unsubscribe(id1, nil); err != nil {
		t.Errorf("Can't unsubscribe: %s", err)
	}

	if len(provider.GetSubscribeIDs()) != 1 {
		t.Errorf("Wrong subscribers count: %d", len(provider.GetSubscribeIDs()))
	}

	// Clear all locks
	if err = provider.SetData("Signal.Cabin.Door.*.IsLocked", []interface{}{
		map[string]interface{}{"Row1.Right.IsLocked": false},
		map[string]interface{}{"Row1.Left.IsLocked": false},
		map[string]interface{}{"Row2.Right.IsLocked": false},
		map[string]interface{}{"Row2.Left.IsLocked": false},
	}, nil); err != nil {
		t.Errorf("Can't set data: %s", err)
	}

	timeout, eventChannel1, eventChannel2 = false, false, false

	for {
		select {
		case _, ok := <-channel1:
			if ok {
				t.Error("Unexpected data received")
			}

			eventChannel1 = true
		case _, ok := <-channel2:
			if !ok {
				t.Error("Channel should not be closed")
			}

			eventChannel2 = true
		case <-time.After(100 * time.Millisecond):
			timeout = true
		}

		if eventChannel1 && eventChannel2 {
			break
		}

		if timeout {
			break
		}
	}

	// Unsubscribes id2
	if err = provider.Unsubscribe(id2, nil); err != nil {
		t.Errorf("Can't unsubscribe: %s", err)
	}

	if len(provider.GetSubscribeIDs()) != 0 {
		t.Errorf("Wrong subscribers count: %d", len(provider.GetSubscribeIDs()))
	}

	// Set all locks
	if err = provider.SetData("Signal.Cabin.Door.*.IsLocked", []interface{}{
		map[string]interface{}{"Row1.Right.IsLocked": true},
		map[string]interface{}{"Row1.Left.IsLocked": true},
		map[string]interface{}{"Row2.Right.IsLocked": true},
		map[string]interface{}{"Row2.Left.IsLocked": true},
	}, nil); err != nil {
		t.Errorf("Can't set data: %s", err)
	}

	timeout, eventChannel1, eventChannel2 = false, false, false

	for {
		select {
		case _, ok := <-channel1:
			if ok {
				t.Error("Unexpected data received")
			}

			eventChannel1 = true
		case _, ok := <-channel2:
			if ok {
				t.Error("Unexpected data received")
			}

			eventChannel2 = true
		case <-time.After(100 * time.Millisecond):
			timeout = true
		}

		if eventChannel1 && eventChannel2 {
			break
		}

		if timeout {
			break
		}
	}
}

func TestPathFilter(t *testing.T) {
	type resultDesc struct {
		path  string
		match bool
	}

	type testDesc struct {
		filter string
		result []resultDesc
	}

	testData := []testDesc{
		{
			filter: "Sensors.Vehicle.Door",
			result: []resultDesc{
				{"Sensors.Vehicle.Door", true},
				{"Sensors.Vehicle.DoorFront", false},
				{"Sensors.Vehicle.DoorRear", false},
			},
		},
		{
			filter: "Sensors.Vehicle.Door",
			result: []resultDesc{
				{"Sensors.Vehicle.Door.Front", true},
				{"Sensors.Vehicle.Door.Rear", true},
			},
		},
		{
			filter: "Sensors.Vehicle.*",
			result: []resultDesc{
				{"Sensors.Vehicles", false},
				{"Sensors.Vehicle.Door.Front", true},
				{"Sensors.Vehicle.Door.Rear", true},
				{"Sensors.Vehicle.Engine", true},
				{"Sensors.Vehicle.Window.Front.Position", true},
				{"Sensors.Vehicle.Window.Rear.Position", true},
				{"Sensors.Engine.Temp", false},
				{"Sensors.Engine.RPM", false},
			},
		},
		{
			filter: "Sensors.Vehicle.*.Front",
			result: []resultDesc{
				{"Sensors.Vehicle.Door.Front", true},
				{"Sensors.Vehicle.Door.Rear", false},
				{"Sensors.Vehicle.Engine", false},
				{"Sensors.Vehicle.Window.Front.Position", true},
				{"Sensors.Vehicle.Window.Rear.Position", false},
			},
		},
	}

	for _, testItem := range testData {
		regexp, err := dataprovider.CreatePathFilter(testItem.filter)
		if err != nil {
			t.Errorf("Can't create regexp from path: %s", err)
			continue
		}

		for _, result := range testItem.result {
			if regexp.Match(result.path) != result.match {
				if result.match {
					t.Errorf("Path %s doesn't match filter %s", result.path, testItem.filter)
				} else {
					t.Errorf("Path %s shouldn't match filter %s", result.path, testItem.filter)
				}
			}
		}
	}
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func arrayToMap(data interface{}) (result map[string]interface{}, err error) {
	// Create map from array
	array, ok := data.([]map[string]interface{})
	if !ok {
		return result, aoserrors.Errorf("wrong data type: %s", reflect.TypeOf(data))
	}

	result = make(map[string]interface{})

	for _, arrayItem := range array {
		for path, value := range arrayItem {
			result[path] = value
		}
	}

	return result, nil
}
