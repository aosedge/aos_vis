// SPDX-License-Identifier: Apache-2.0
//
// Copyright 2021 Renesas Inc.
// Copyright 2021 EPAM Systems Inc.
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

package permissionprovider_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	pb "gitpct.epam.com/epmd-aepr/aos_common/api/iamanager"

	"aos_vis/config"
	"aos_vis/permissionprovider"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

type testServer struct {
	grpcServer  *grpc.Server
	permissions map[string]map[string]string
}

/*******************************************************************************
 * Consts
 ******************************************************************************/

const serverURL = "localhost:8090"
const visFunctionalServerID = "vis"
const secret = "secret_ID"

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

/*******************************************************************************
 * Main
 ******************************************************************************/

func TestGetPermissions(t *testing.T) {
	origPermissions := make(map[string]string)
	origPermissions["*"] = "rw"

	server, err := newTestServer(serverURL)
	if err != nil {
		t.Fatalf("Can't create test server: %s", err)
	}
	defer server.close()

	server.SetPermissions(secret, origPermissions)

	permissionProvider, err := permissionprovider.New(&config.Config{PermissionServerURL: serverURL}, true)
	if err != nil {
		t.Fatalf("Can't create permission provider: %s", err)
	}
	defer permissionProvider.Close()

	permissions, err := permissionProvider.GetVisPermissionByToken(secret)
	if err != nil {
		t.Errorf("Can't get permissions: %s", err)
	}

	if !reflect.DeepEqual(origPermissions, permissions) {
		t.Errorf("Incorrect permissions: %s", err)
	}
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func newTestServer(url string) (server *testServer, err error) {
	server = &testServer{permissions: make(map[string]map[string]string)}

	listener, err := net.Listen("tcp", url)
	if err != nil {
		return nil, err
	}
	server.grpcServer = grpc.NewServer()

	pb.RegisterIAManagerPublicServer(server.grpcServer, server)

	go server.grpcServer.Serve(listener)

	return server, nil
}

func (server *testServer) close() (err error) {
	if server.grpcServer != nil {
		server.grpcServer.Stop()
	}

	return nil
}

func (server *testServer) GetPermissions(ctx context.Context, req *pb.GetPermissionsReq) (rsp *pb.GetPermissionsRsp, err error) {
	rsp = &pb.GetPermissionsRsp{}

	if req.FunctionalServerId != visFunctionalServerID {
		return rsp, fmt.Errorf("incorrect functional server ID: %s", req.FunctionalServerId)
	}

	servicePermissions, ok := server.permissions[req.Secret]
	if !ok {
		return rsp, fmt.Errorf("secret not found")
	}

	rsp.Permissions = &pb.Permissions{Permissions: servicePermissions}

	return rsp, nil
}

func (server *testServer) SetPermissions(secret string, permissions map[string]string) {
	server.permissions[secret] = permissions
}
