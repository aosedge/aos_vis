// SPDX-License-Identifier: Apache-2.0
//
// Copyright 2019 Renesas Inc.
// Copyright 2019 EPAM Systems Inc.
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

package dbusclient

import (
	"encoding/json"

	"github.com/godbus/dbus"
	log "github.com/sirupsen/logrus"
)

const (
	objectPath    = "/com/aos/servicemanager/vis"
	interfaceName = "com.aos.servicemanager.vis"
)

// GetVisPermissionByToken dbus call GetPermission
func GetVisPermissionByToken(token string, useSystemBus bool) (permissions map[string]string, err error) {
	var permissionJSON string
	var dbusErr string
	var conn *dbus.Conn

	if useSystemBus == true {
		conn, err = dbus.SystemBus()
	} else {
		conn, err = dbus.SessionBus()
	}

	if err != nil {
		return permissions, err
	}

	obj := conn.Object(interfaceName, objectPath)

	if err = obj.Call(interfaceName+".GetPermission", 0, token).Store(&permissionJSON, &dbusErr); err != nil {
		return permissions, err
	}

	if err = json.Unmarshal([]byte(permissionJSON), &permissions); err != nil {
		return permissions, err
	}

	log.WithFields(log.Fields{"token": token, "permissions": permissions}).Debug("Get permissions")

	return permissions, nil
}
