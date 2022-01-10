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

package visserver

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aoscloud/aos_common/aoserrors"
	"github.com/aoscloud/aos_common/visprotocol"
	"github.com/aoscloud/aos_common/wsserver"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"github.com/aoscloud/aos_vis/config"
	"github.com/aoscloud/aos_vis/dataprovider"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// PermissionProvider interface to get permissions.
type PermissionProvider interface {
	GetVisPermissionByToken(token string) (permissions map[string]string, err error)
}

/*******************************************************************************
 * Consts
 ******************************************************************************/

// VIS actions.
const (
	ActionGet            = "get"
	ActionSet            = "set"
	ActionAuth           = "authorize"
	ActionSubscribe      = "subscribe"
	ActionUnsubscribe    = "unsubscribe"
	ActionUnsubscribeAll = "unsubscribeAll"
	ActionSubscription   = "subscription"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// Server update manager server structure.
type Server struct {
	sync.Mutex
	wsServer           *wsserver.Server
	dataProvider       *dataprovider.DataProvider
	clients            map[*wsserver.Client]*clientInfo
	permissionProvider PermissionProvider
}

type clientInfo struct {
	authInfo           *dataprovider.AuthInfo
	subscribeChannels  map[uint64]<-chan interface{}
	dataProvider       *dataprovider.DataProvider
	wsClient           *wsserver.Client
	permissionProvider PermissionProvider
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// New creates new Web socket server.
func New(config *config.Config, permissionProvider PermissionProvider) (server *Server, err error) {
	log.Debug("Create VIS server")

	server = &Server{clients: make(map[*wsserver.Client]*clientInfo), permissionProvider: permissionProvider}

	if server.dataProvider, err = dataprovider.New(config); err != nil {
		return nil, aoserrors.Wrap(err)
	}

	if server.wsServer, err = wsserver.New("VIS", config.ServerURL, config.VISCert, config.VISKey, server); err != nil {
		return nil, aoserrors.Wrap(err)
	}

	return server, nil
}

// Close closes web socket server and all connections.
func (server *Server) Close() {
	server.Lock()
	defer server.Unlock()

	server.wsServer.Close()
	server.dataProvider.Close()
}

// ClientConnected connect client notification.
func (server *Server) ClientConnected(client *wsserver.Client) {
	server.Lock()
	defer server.Unlock()
	log.Info("ClientConnected")

	server.clients[client] = &clientInfo{
		authInfo:          &dataprovider.AuthInfo{},
		subscribeChannels: make(map[uint64]<-chan interface{}),
		dataProvider:      server.dataProvider,
		wsClient:          client,
	}

	log.Info("GetPermissionProvider")

	server.clients[client].permissionProvider = server.GetPermissionProvider()
}

// ClientDisconnected disconnect client notification.
func (server *Server) ClientDisconnected(client *wsserver.Client) {
	server.Lock()
	defer server.Unlock()

	delete(server.clients, client)
}

// ProcessMessage proccess incoming messages.
func (server *Server) ProcessMessage(
	wsClient *wsserver.Client, messageType int, message []byte) (response []byte, err error) {
	server.Lock()
	defer server.Unlock()

	if messageType != websocket.TextMessage {
		return nil, aoserrors.New("incoming message in unsupported format")
	}

	client, ok := server.clients[wsClient]
	if !ok {
		return nil, aoserrors.New("message from unknown client")
	}

	var header visprotocol.MessageHeader

	if err = json.Unmarshal(message, &header); err != nil {
		return nil, aoserrors.Wrap(err)
	}

	var responseItf interface{}

	switch string(header.Action) {
	case ActionGet:
		responseItf, err = client.processGetRequest(message)

	case ActionSet:
		responseItf, err = client.processSetRequest(message)

	case ActionAuth:
		responseItf, err = client.processAuthRequest(message)

	case ActionSubscribe:
		responseItf, err = client.processSubscribeRequest(message)

	case ActionUnsubscribe:
		responseItf, err = client.processUnsubscribeRequest(message)

	case ActionUnsubscribeAll:
		responseItf, err = client.processUnsubscribeAllRequest(message)

	default:
		err = aoserrors.Errorf("unsupported action type: %s", header.Action)
	}

	if err != nil {
		return nil, aoserrors.Wrap(err)
	}

	if response, err = json.Marshal(responseItf); err != nil {
		return nil, aoserrors.Wrap(err)
	}

	return response, nil
}

// GetPermissionProvider returns permission provider interface.
func (server *Server) GetPermissionProvider() (permissionProvider PermissionProvider) {
	return server.permissionProvider
}

/*******************************************************************************
 * Private
 ******************************************************************************/

// process Get request.
func (client *clientInfo) processGetRequest(requestJSON []byte) (response *visprotocol.GetResponse, err error) {
	var request visprotocol.GetRequest

	if err = json.Unmarshal(requestJSON, &request); err != nil {
		return nil, aoserrors.Wrap(err)
	}

	response = &visprotocol.GetResponse{
		MessageHeader: request.MessageHeader,
		Timestamp:     getCurTime(),
	}

	vehicleData, err := client.dataProvider.GetData(request.Path, client.authInfo)
	if err != nil {
		response.Error = createErrorInfo(err)
		return response, nil
	}

	response.Value = vehicleData

	return response, nil
}

// process Set request.
func (client *clientInfo) processSetRequest(requestJSON []byte) (response *visprotocol.SetResponse, err error) {
	var request visprotocol.SetRequest

	if err = json.Unmarshal(requestJSON, &request); err != nil {
		return nil, aoserrors.Wrap(err)
	}

	response = &visprotocol.SetResponse{
		MessageHeader: request.MessageHeader,
		Timestamp:     getCurTime(),
	}

	if err = client.dataProvider.SetData(request.Path, request.Value, client.authInfo); err != nil {
		response.Error = createErrorInfo(err)
		return response, nil
	}

	return response, nil
}

// process Auth request.
func (client *clientInfo) processAuthRequest(requestJSON []byte) (response *visprotocol.AuthResponse, err error) {
	var request visprotocol.AuthRequest

	if err = json.Unmarshal(requestJSON, &request); err != nil {
		return nil, aoserrors.Wrap(err)
	}

	response = &visprotocol.AuthResponse{
		MessageHeader: request.MessageHeader,
	}

	if request.Tokens.Authorization == "" {
		response.Error = createErrorInfo(aoserrors.New("empty token authorization"))
		return response, nil
	}

	if client.authInfo.Permissions,
		err = client.permissionProvider.GetVisPermissionByToken(request.Tokens.Authorization); err != nil {
		log.Error("err: ", err)

		response.Error = createErrorInfo(aoserrors.New("service not authorized"))

		return response, nil
	}

	client.authInfo.IsAuthorized = true
	response.TTL = 10000

	return response, nil
}

// process Subscribe request.
func (client *clientInfo) processSubscribeRequest(requestJSON []byte) (responseItf interface{}, err error) {
	var request visprotocol.SubscribeRequest

	if err = json.Unmarshal(requestJSON, &request); err != nil {
		return nil, aoserrors.Wrap(err)
	}

	if request.Filters != "" {
		log.Warn("Filter currently not implemented. Filters will be ignored")
	}

	response := visprotocol.SubscribeResponse{
		MessageHeader: request.MessageHeader,
		Timestamp:     getCurTime(),
	}

	id, channel, err := client.dataProvider.Subscribe(request.Path, client.authInfo)
	if err != nil {
		response.Error = createErrorInfo(err)
		return &response, nil
	}

	log.WithFields(log.Fields{"path": request.Path, "id": id}).Debug("Register subscription")

	response.SubscriptionID = strconv.FormatUint(id, 10)

	client.subscribeChannels[id] = channel
	go client.processSubscribeChannel(id, channel)

	return &response, nil
}

// process Unsubscribe request.
func (client *clientInfo) processUnsubscribeRequest(requestJSON []byte) (responseItf interface{}, err error) {
	var request visprotocol.UnsubscribeRequest

	if err = json.Unmarshal(requestJSON, &request); err != nil {
		return nil, aoserrors.Wrap(err)
	}

	response := visprotocol.UnsubscribeResponse{
		MessageHeader:  request.MessageHeader,
		SubscriptionID: request.SubscriptionID,
		Timestamp:      getCurTime(),
	}

	subscribeID, err := strconv.ParseUint(request.SubscriptionID, 10, 64)
	if err != nil {
		response.Error = createErrorInfo(err)
		return &response, nil
	}

	if err = client.dataProvider.Unsubscribe(subscribeID, client.authInfo); err != nil {
		response.Error = createErrorInfo(err)
		return &response, nil
	}

	delete(client.subscribeChannels, subscribeID)

	log.WithFields(log.Fields{"id": request.SubscriptionID}).Debug("Unregister subscription")

	return &response, nil
}

// process UnsubscribeAll request.
func (client *clientInfo) processUnsubscribeAllRequest(requestJSON []byte) (responseItf interface{}, err error) {
	var request visprotocol.UnsubscribeAllRequest

	if err = json.Unmarshal(requestJSON, &request); err != nil {
		return nil, aoserrors.Wrap(err)
	}

	response := visprotocol.UnsubscribeAllResponse{
		MessageHeader: request.MessageHeader,
		Timestamp:     getCurTime(),
	}

	if err = client.unsubscribeAll(); err != nil {
		response.Error = createErrorInfo(err)
		return &response, nil
	}

	return &response, nil
}

func (client *clientInfo) processSubscribeChannel(id uint64, channel <-chan interface{}) {
	for {
		data, more := <-channel
		if more {
			subscriptionID := strconv.FormatUint(id, 10)

			notification := visprotocol.SubscriptionNotification{
				Action:         ActionSubscription,
				SubscriptionID: subscriptionID,
				Value:          data,
				Timestamp:      getCurTime(),
			}

			notificationJSON, err := json.Marshal(notification)
			if err != nil {
				log.Errorf("Can't marshal subscription notification: %s", err)
			}

			if notificationJSON != nil {
				if err := client.wsClient.SendMessage(websocket.TextMessage, notificationJSON); err != nil {
					log.Errorf("Can't send message: %s", err)
				}
			}
		} else {
			log.WithField("subscribeID", id).Debug("Subscription closed")
			return
		}
	}
}

func (client *clientInfo) unsubscribeAll() (err error) {
	for subscribeID := range client.subscribeChannels {
		if localErr := client.dataProvider.Unsubscribe(subscribeID, client.authInfo); localErr != nil {
			err = localErr
		}
	}

	client.subscribeChannels = make(map[uint64]<-chan interface{})

	return aoserrors.Wrap(err)
}

func createErrorInfo(err error) (errorInfo *visprotocol.ErrorInfo) {
	if err == nil {
		return nil
	}

	errorInfo = &visprotocol.ErrorInfo{Message: err.Error()}

	switch {
	case strings.Contains(strings.ToLower(err.Error()), "not found") ||
		strings.Contains(strings.ToLower(err.Error()), "not exist"):
		errorInfo.Number = 404
	case strings.Contains(strings.ToLower(err.Error()), "not authorized"):
		errorInfo.Number = 401
	case strings.Contains(strings.ToLower(err.Error()), "not have permissions"):
		errorInfo.Number = 403
	default:
		errorInfo.Number = 400
	}

	return errorInfo
}

func getCurTime() int64 {
	return time.Now().UnixNano() / 1000000 // nolint:gomnd
}
