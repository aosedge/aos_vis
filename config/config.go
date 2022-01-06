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

// Package config provides set of API to provide aos configuration
package config

import (
	"encoding/json"
	"os"

	"github.com/aoscloud/aos_common/aoserrors"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// Config instance
type Config struct {
	ServerURL           string          `json:"serverURL"`
	CACert              string          `json:"caCert"`
	VISCert             string          `json:"VISCert"`
	VISKey              string          `json:"VISKey"`
	Adapters            []AdapterConfig `json:"adapters"`
	PermissionServerURL string          `json:"permissionServerURL"`
}

// AdapterConfig adapter configuration
type AdapterConfig struct {
	Plugin   string          `json:"plugin"`
	Disabled bool            `json:"disabled"`
	Params   json.RawMessage `json:"params"`
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// New creates new config object
func New(fileName string) (config *Config, err error) {
	file, err := os.Open(fileName)
	if err != nil {
		return config, aoserrors.Wrap(err)
	}

	config = &Config{}

	decoder := json.NewDecoder(file)
	if err = decoder.Decode(config); err != nil {
		return config, aoserrors.Wrap(err)
	}

	return config, nil
}
