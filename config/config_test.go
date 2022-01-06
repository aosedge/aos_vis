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

package config_test

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"

	"github.com/aoscloud/aos_common/aoserrors"

	"github.com/aoscloud/aos_vis/config"
)

/*******************************************************************************
 * Private
 ******************************************************************************/

func createConfigFile() (err error) {
	configContent := `{
"ServerUrl": "localhost:443",
"CACert": "/etc/ssl/certs/rootCA.crt",
"VISCert": "wwwivi.crt.pem",
"VISKey": "wwwivi.key.pem",
"Adapters":[{
		"Plugin": "test1",
		"Disabled": true
	}, {
		"Plugin": "test2"
	}, {
		"Plugin": "test3"
	}],
"PermissionServerURL": "aosiam:8090"
}`

	if err := ioutil.WriteFile(path.Join("tmp", "visconfig.json"), []byte(configContent), 0o644); err != nil {
		return aoserrors.Wrap(err)
	}

	return nil
}

func setup() (err error) {
	if err := os.MkdirAll("tmp", 0o755); err != nil {
		return aoserrors.Wrap(err)
	}

	if err = createConfigFile(); err != nil {
		return aoserrors.Wrap(err)
	}

	return nil
}

func cleanup() (err error) {
	if err := os.RemoveAll("tmp"); err != nil {
		return aoserrors.Wrap(err)
	}

	return nil
}

/*******************************************************************************
 * Main
 ******************************************************************************/

func TestMain(m *testing.M) {
	if err := setup(); err != nil {
		log.Fatalf("Error creating service images: %s", err)
	}

	ret := m.Run()

	if err := cleanup(); err != nil {
		log.Fatalf("Error cleaning up: %s", err)
	}

	os.Exit(ret)
}

/*******************************************************************************
 * Tests
 ******************************************************************************/

func TestGetCredentials(t *testing.T) {
	config, err := config.New("tmp/visconfig.json")
	if err != nil {
		t.Fatalf("Error opening config file: %s", err)
	}

	if config.ServerURL != "localhost:443" {
		t.Errorf("Wrong ServerURL value: %s", config.ServerURL)
	}

	if config.VISCert != "wwwivi.crt.pem" {
		t.Errorf("Wrong VISCert value: %s", config.VISCert)
	}

	if config.VISKey != "wwwivi.key.pem" {
		t.Errorf("Wrong VISKey value: %s", config.VISKey)
	}

	if config.CACert != "/etc/ssl/certs/rootCA.crt" {
		t.Errorf("Wrong CACert value: %s", config.CACert)
	}
}

func TestAdapters(t *testing.T) {
	config, err := config.New("tmp/visconfig.json")
	if err != nil {
		t.Fatalf("Error opening config file: %s", err)
	}

	if len(config.Adapters) != 3 {
		t.Errorf("Wrong adapters len: %d", len(config.Adapters))
	}

	if config.Adapters[0].Plugin != "test1" || config.Adapters[1].Plugin != "test2" || config.Adapters[2].Plugin != "test3" {
		t.Error("Wrong adapter name")
	}

	if config.Adapters[0].Disabled != true || config.Adapters[1].Disabled != false || config.Adapters[2].Disabled != false {
		t.Error("Wrong disable value")
	}
}

func TestPermissionServerURL(t *testing.T) {
	config, err := config.New("tmp/visconfig.json")
	if err != nil {
		t.Fatalf("Error opening config file: %s", err)
	}

	if config.PermissionServerURL != "aosiam:8090" {
		t.Errorf("Wrong PermissionServerURL value: %s", config.ServerURL)
	}
}
