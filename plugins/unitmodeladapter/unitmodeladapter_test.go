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

package unitmodeladapter_test

import (
	"encoding/json"
	"os"
	"path"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/aosedge/aos_vis/plugins/unitmodeladapter"
)

/*******************************************************************************
 * Consts
 ******************************************************************************/

const unitModelVISPath = "Attribute.Aos.UnitModel"

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

	tmpDir, err = os.MkdirTemp("", "vis_")
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
	adapter, err := unitmodeladapter.New(generateConfig(unitModelVISPath, path.Join(tmpDir, "unit_model.txt")))
	if err != nil {
		t.Fatalf("Can't create adapter: %s", err)
	}
	defer adapter.Close()

	if name := adapter.GetName(); name != "unitmodeladapter" {
		t.Errorf("Wrong adapter name: %s", name)
	}
}

func TestGetUnitModel(t *testing.T) {
	unitModelFile := path.Join(tmpDir, "unitmodel.txt")
	originUnitModel := "TEST_UNIT_MODEL"

	if err := os.WriteFile(unitModelFile, []byte(originUnitModel), 0o600); err != nil {
		t.Fatalf("Can't create unit model file: %s", err)
	}

	adapter, err := unitmodeladapter.New(generateConfig(unitModelVISPath, unitModelFile))
	if err != nil {
		t.Fatalf("Can't create adapter: %s", err)
	}
	defer adapter.Close()

	data, err := adapter.GetData([]string{unitModelVISPath})
	if err != nil {
		t.Fatalf("Can't get data: %s", err)
	}

	if _, ok := data[unitModelVISPath]; !ok {
		t.Fatal("unit model not found in data")
	}

	unitModel, ok := data[unitModelVISPath].(string)
	if !ok {
		t.Fatal("Wrong unit model data type")
	}

	if unitModel != originUnitModel {
		t.Errorf("Wrong unit model value: %s", unitModel)
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
