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

package permissionprovider

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/aosedge/aos_common/aoserrors"
	pb "github.com/aosedge/aos_common/api/iamanager/v4"
	"github.com/aosedge/aos_common/utils/cryptutils"

	"github.com/aosedge/aos_vis/config"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// PermissionProvider vis permission provider.
type PermissionProvider struct {
	serverURL     string
	rootCert      string
	insecure      bool
	cryptoContext *cryptutils.CryptoContext
	iamClient     pb.IAMPublicPermissionsServiceClient
	connection    *grpc.ClientConn
}

/*******************************************************************************
 * Consts
 ******************************************************************************/

const (
	iamRequestTimeout = 30 * time.Second
)

const visFunctionalServerID = "vis"

/*******************************************************************************
 * Public
 ******************************************************************************/

// New creates new permission provider.
func New(config *config.Config, insecure bool) (provider *PermissionProvider, err error) {
	provider = &PermissionProvider{
		serverURL: config.PermissionServerURL,
		rootCert:  config.CACert, iamClient: nil, insecure: insecure, connection: nil,
	}

	if provider.cryptoContext, err = cryptutils.NewCryptoContext(config.CACert); err != nil {
		return nil, aoserrors.Wrap(err)
	}

	return provider, nil
}

// GetVisPermissionByToken get vis permission by token.
func (provider *PermissionProvider) GetVisPermissionByToken(token string) (permissions map[string]string, err error) {
	if provider.connection == nil {
		if err = provider.connect(); err != nil {
			return permissions, err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), iamRequestTimeout)
	defer cancel()

	req := &pb.PermissionsRequest{Secret: token, FunctionalServerId: visFunctionalServerID}

	response, err := provider.iamClient.GetPermissions(ctx, req)
	if err != nil {
		return permissions, aoserrors.Wrap(err)
	}

	return response.GetPermissions().GetPermissions(), nil
}

// Close close connection with permission provider grpc server.
func (provider *PermissionProvider) Close() {
	if provider.connection != nil {
		provider.connection.Close()
	}
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func (provider *PermissionProvider) connect() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), iamRequestTimeout)
	defer cancel()

	var secureOpt grpc.DialOption

	if provider.insecure {
		secureOpt = grpc.WithTransportCredentials(insecure.NewCredentials())
	} else {
		tlsConfig, err := provider.cryptoContext.GetClientTLSConfig()
		if err != nil {
			return aoserrors.Wrap(err)
		}

		secureOpt = grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
	}

	if provider.connection, err = grpc.DialContext(ctx, provider.serverURL, secureOpt, grpc.WithBlock()); err != nil {
		return aoserrors.Wrap(err)
	}

	provider.iamClient = pb.NewIAMPublicPermissionsServiceClient(provider.connection)

	return nil
}
