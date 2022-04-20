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

package wsclient

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"github.com/aoscloud/aos_common/aoserrors"
	"github.com/aoscloud/aos_common/utils/cryptutils"
)

/***********************************************************************************************************************
 * Consts
 **********************************************************************************************************************/

const (
	defaultWebsocketTimeout = 120 * time.Second
	errorChannelSize        = 1
)

/***********************************************************************************************************************
 * Types
 **********************************************************************************************************************/

// Client VIS client object.
type Client struct {
	ErrorChannel chan error

	name           string
	messageHandler func([]byte)
	connection     *websocket.Conn
	requests       sync.Map
	sync.Mutex
	isConnected       bool
	disconnectChannel chan bool
	wsDialer          websocket.Dialer
	clientParam       ClientParam
	cryptoContext     *cryptutils.CryptoContext
}

// ClientParam client parameters.
type ClientParam struct {
	CaCertFile       string
	WebSocketTimeout time.Duration
}

type requestParam struct {
	id         interface{}
	idField    string
	rspChannel chan bool
	rsp        interface{}
}

/***********************************************************************************************************************
 * Public
 **********************************************************************************************************************/

// New creates new ws client.
func New(name string, clientParam ClientParam, messageHandler func([]byte)) (client *Client, err error) {
	log.WithFields(log.Fields{"client": name}).Debug("New ws client")

	client = &Client{
		name:              name,
		messageHandler:    messageHandler,
		ErrorChannel:      make(chan error, errorChannelSize),
		disconnectChannel: make(chan bool),
		clientParam:       clientParam,
	}

	// Check if system root certificate override is active and if so update tls config with custom CA
	if len(clientParam.CaCertFile) > 0 {
		cryptoContext, err := cryptutils.NewCryptoContext(clientParam.CaCertFile)
		if err != nil {
			return nil, aoserrors.Wrap(err)
		}

		client.cryptoContext = cryptoContext

		if client.wsDialer.TLSClientConfig, err = cryptoContext.GetClientTLSConfig(); err != nil {
			return nil, aoserrors.Wrap(err)
		}

		log.WithFields(log.Fields{
			"client": client.name,
			"caCert": clientParam.CaCertFile,
		}).Debug("Updating TLS config based on caCert")
	}

	if clientParam.WebSocketTimeout > 0 {
		client.clientParam.WebSocketTimeout = clientParam.WebSocketTimeout
	} else {
		client.clientParam.WebSocketTimeout = defaultWebsocketTimeout
	}

	return client, nil
}

// Connect connects to ws server.
func (client *Client) Connect(url string) (err error) {
	client.Lock()
	defer client.Unlock()

	log.WithFields(log.Fields{
		"client":    client.name,
		"url":       url,
		"wsTimeout": client.clientParam.WebSocketTimeout,
	}).Debug("Connect to server")

	if client.isConnected {
		return aoserrors.Errorf("client %s already connected", client.name)
	}

	connection, _, err := client.wsDialer.Dial(url, nil)
	if err != nil {
		return aoserrors.Wrap(err)
	}

	client.connection = connection

	client.isConnected = true

	go client.processMessages()

	return nil
}

// Disconnect disconnects from ws server.
func (client *Client) Disconnect() (err error) {
	client.Lock()

	if !client.isConnected {
		client.Unlock()

		return nil
	}

	log.WithFields(log.Fields{"client": client.name}).Debug("Disconnect")

	client.isConnected = false

	if e := client.connection.SetWriteDeadline(time.Now().Add(client.clientParam.WebSocketTimeout)); e != nil {
		log.Errorf("Can't set write deadline timeout: %s", e)
		err = e
	}

	if e := client.connection.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); e != nil {
		log.Errorf("Can't send close message: %s", e)
		err = e
	}

	if e := client.connection.Close(); e != nil {
		log.Errorf("Can't close web socket: %s", e)
		err = e
	}

	client.Unlock()

	select {
	case <-client.disconnectChannel:

	case <-time.After(1 * time.Second):
		log.Warn("Waiting for disconnect timeout")
	}

	return aoserrors.Wrap(err)
}

// GenerateRequestID generates unique request ID.
func GenerateRequestID() (requestID string) {
	return uuid.New().String()
}

// IsConnected returns true if connected to ws server.
func (client *Client) IsConnected() (result bool) {
	client.Lock()
	defer client.Unlock()

	return client.isConnected
}

// Close closes ws client.
func (client *Client) Close() (err error) {
	log.WithFields(log.Fields{"client": client.name}).Info("Close ws client")

	if disconnectErr := aoserrors.Wrap(client.Disconnect()); disconnectErr != nil {
		if err == nil {
			err = aoserrors.Wrap(disconnectErr)
		}
	}

	if client.cryptoContext != nil {
		if contextErr := client.cryptoContext.Close(); contextErr != nil {
			if err == nil {
				err = aoserrors.Wrap(contextErr)
			}
		}
	}

	return err
}

// SendRequest sends request and waits for response.
func (client *Client) SendRequest(idField string, idValue interface{}, req interface{}, rsp interface{}) (err error) {
	requestID := reflect.ValueOf(req).Elem()

	if requestID.Kind() == reflect.Ptr {
		requestID = requestID.Elem()
	}

	for _, field := range strings.Split(idField, ".") {
		requestID = requestID.FieldByName(field)
		if !requestID.IsValid() {
			return aoserrors.New("ID is invalid")
		}
	}

	param := requestParam{id: idValue, idField: idField, rspChannel: make(chan bool), rsp: rsp}
	client.requests.Store(param.id, param)

	defer client.requests.Delete(param.id)

	if err = client.SendMessage(req); err != nil {
		return aoserrors.Wrap(err)
	}

	// Wait response or timeout
	select {
	case <-time.After(client.clientParam.WebSocketTimeout):
		return aoserrors.New("wait response timeout")

	case _, ok := <-param.rspChannel:
		if !ok {
			return aoserrors.New("response channel is closed")
		}
	}

	return nil
}

// SendMessage sends message without waiting for response.
func (client *Client) SendMessage(message interface{}) (err error) {
	client.Lock()
	defer client.Unlock()

	if !client.isConnected {
		return aoserrors.New("client is disconnected")
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		return aoserrors.Wrap(err)
	}

	log.WithFields(log.Fields{"client": client.name, "message": string(messageJSON)}).Debug("Send message")

	if err := client.connection.SetWriteDeadline(time.Now().Add(client.clientParam.WebSocketTimeout)); err != nil {
		log.WithFields(log.Fields{"client": client.name}).Debugf("Can't set write deadline timeout: %s", err)

		client.connection.Close()

		return aoserrors.Wrap(err)
	}

	if err = client.connection.WriteMessage(websocket.TextMessage, messageJSON); err != nil {
		log.WithFields(log.Fields{"client": client.name}).Debugf("Send message error: %s", err)
		client.connection.Close()

		return aoserrors.Wrap(err)
	}

	return nil
}

/***********************************************************************************************************************
 * Private
 **********************************************************************************************************************/

func (client *Client) processMessages() {
	for {
		_, message, err := client.connection.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure) &&
				!strings.Contains(err.Error(), "use of closed network connection") {
				log.WithFields(log.Fields{"client": client.name}).Errorf("Receive message error: %s", err)
			}

			client.disconnect(err)

			return
		}

		log.WithFields(log.Fields{"client": client.name, "message": string(message)}).Debug("Receive message")

		rspFound := client.findRequestID(message)

		if client.messageHandler != nil && !rspFound {
			client.messageHandler(message)
		}
	}
}

func (client *Client) findRequestID(message []byte) (found bool) {
	client.requests.Range(func(key, value interface{}) bool {
		param, ok := value.(requestParam)
		if !ok {
			return true
		}

		if err := json.Unmarshal(message, param.rsp); err != nil {
			return true
		}

		requestID := reflect.ValueOf(param.rsp).Elem()

		for _, field := range strings.Split(param.idField, ".") {
			requestID = requestID.FieldByName(field)
			if !requestID.IsValid() {
				return true
			}
		}

		if requestID.Kind() == reflect.Ptr {
			requestID = requestID.Elem()
		}

		if key == requestID.Interface() {
			client.requests.Delete(param.id)

			param.rspChannel <- true
			found = true

			return false
		}

		return true
	})

	return found
}

func (client *Client) disconnect(err error) {
	client.Lock()
	defer client.Unlock()

	if client.isConnected {
		log.WithFields(log.Fields{"client": client.name}).Debug("Remote disconnect")

		client.connection.Close()
		client.isConnected = false

		client.ErrorChannel <- err
	} else {
		client.disconnectChannel <- true
	}
}
