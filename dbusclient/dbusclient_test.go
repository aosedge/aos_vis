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

package dbusclient_test

import (
	"os"
	"testing"

	"github.com/godbus/dbus"
	log "github.com/sirupsen/logrus"

	"aos_vis/dbusclient"
)

/*******************************************************************************
 * Init
 ******************************************************************************/

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		TimestampFormat:  "2006-01-02 15:04:05.000",
		FullTimestamp:    true})
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
}

type dbusInterface struct {
}

func (GetPermission dbusInterface) GetPermission(token string) (string, string, *dbus.Error) {
	return `{"*": "rw", "123": "rw"}`, "OK", nil
}

/*******************************************************************************
 * Main
 ******************************************************************************/

func TestMain(m *testing.M) {
	conn, err := dbus.SessionBus()
	if err != nil {
		log.Fatalf("Can't create session connection: %v", err)
	}

	reply, err := conn.RequestName("com.aos.servicemanager.vis", dbus.NameFlagDoNotQueue)
	if err != nil {
		log.Fatal("Can't request name")
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		log.Fatal("Name already taken")
	}

	server := dbusInterface{}
	conn.Export(server, "/com/aos/servicemanager/vis", "com.aos.servicemanager.vis")

	ret := m.Run()

	os.Exit(ret)
}

func TestDBUS(t *testing.T) {
	permission, err := dbusclient.GetVisPermissionByToken("APPID", false)
	if err != nil {
		t.Fatalf("Can't make D-Bus call: %s", err)
	}

	if len(permission) != 2 {
		t.Fatal("Permission list length !=2")
	}

	if permission["*"] != "rw" {
		t.Fatal("Incorrect permissions")
	}
}
