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

package vinadapter_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/aoscloud/aos_vis/plugins/vinadapter"
)

/*******************************************************************************
 * Consts
 ******************************************************************************/

const vinVISPath = "Attribute.Vehicle.VehicleIdentification.VIN"

/*******************************************************************************
 * Vars
 ******************************************************************************/

var tmpDir string

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
	var err error

	tmpDir, err = ioutil.TempDir("", "vis_")
	if err != nil {
		log.Fatalf("Error creating tmp dir: %s", err)
	}

	ret := m.Run()

	if err := os.RemoveAll(tmpDir); err != nil {
		log.Fatalf("Error removing tmp dir: %s", err)
	}

	os.Exit(ret)
}

/*******************************************************************************
 * Tests
 ******************************************************************************/

func TestGetName(t *testing.T) {
	adapter, err := vinadapter.New(generateConfig(vinVISPath, path.Join(tmpDir, "vin.txt")))
	if err != nil {
		t.Fatalf("Can't create adapter: %s", err)
	}
	defer adapter.Close()

	if name := adapter.GetName(); name != "vinadapter" {
		t.Errorf("Wrong adapter name: %s", name)
	}
}

func TestGenerateVIN(t *testing.T) {
	vinFile := path.Join(tmpDir, "vin.txt")
	if err := os.RemoveAll(vinFile); err != nil {
		t.Fatalf("Can't remove VIN file: %s", err)
	}

	adapter, err := vinadapter.New(generateConfig(vinVISPath, vinFile))
	if err != nil {
		t.Fatalf("Can't create adapter: %s", err)
	}
	defer adapter.Close()

	data, err := adapter.GetData([]string{vinVISPath})
	if err != nil {
		t.Fatalf("Can't get data: %s", err)
	}

	if _, ok := data[vinVISPath]; !ok {
		t.Fatal("VIN not found in data")
	}

	vin, ok := data[vinVISPath].(string)
	if !ok {
		t.Fatal("Wrong VIN data type")
	}

	generatedVIN, err := ioutil.ReadFile(vinFile)
	if err != nil {
		t.Fatalf("Can't read VIN file: %s", err)
	}

	if vin != string(generatedVIN) {
		t.Errorf("Wrong VIN value: %s", vin)
	}
}

func TestExistingVIN(t *testing.T) {
	vinFile := path.Join(tmpDir, "vin.txt")
	originVin := "TEST_VIN"

	if err := ioutil.WriteFile(vinFile, []byte(originVin), 0o644); err != nil {
		t.Fatalf("Can't create VIN file: %s", err)
	}

	adapter, err := vinadapter.New(generateConfig(vinVISPath, vinFile))
	if err != nil {
		t.Fatalf("Can't create adapter: %s", err)
	}
	defer adapter.Close()

	data, err := adapter.GetData([]string{vinVISPath})
	if err != nil {
		t.Fatalf("Can't get data: %s", err)
	}

	if _, ok := data[vinVISPath]; !ok {
		t.Fatal("VIN not found in data")
	}

	vin, ok := data[vinVISPath].(string)
	if !ok {
		t.Fatal("Wrong VIN data type")
	}

	if vin != originVin {
		t.Errorf("Wrong VIN value: %s", vin)
	}
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func generateConfig(visPath, filePath string) (config []byte) {
	type adapterConfig struct {
		VISPath  string `json:"visPath"`
		FilePath string `json:"filePath"`
	}

	var err error

	if config, err = json.Marshal(&adapterConfig{VISPath: visPath, FilePath: filePath}); err != nil {
		log.Fatalf("Can't marshal config: %s", err)
	}

	return config
}
