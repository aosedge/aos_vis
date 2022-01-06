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

package visserver_test

import (
	"bytes"
	"encoding/json"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aoscloud/aos_common/aoserrors"
	"github.com/aoscloud/aos_common/visprotocol"
	"github.com/aoscloud/aos_common/wsclient"
	log "github.com/sirupsen/logrus"

	"github.com/aoscloud/aos_vis/config"
	"github.com/aoscloud/aos_vis/dataprovider"
	"github.com/aoscloud/aos_vis/visserver"
)

const (
	serverURL = "wss://localhost:443"
	caCert    = "../data/rootCA.pem"
)

type permissionProvider struct{}

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

func (provider *permissionProvider) GetVisPermissionByToken(token string) (permissions map[string]string, err error) {
	permission := make(map[string]string)
	permission["Signal.*"] = "rw"

	return permission, nil
}

/*******************************************************************************
 * Main
 ******************************************************************************/

func TestMain(m *testing.M) {
	permissionProvider := permissionProvider{}

	configJSON := `{
		"VISCert": "../data/wwwivi.crt.pem",
		"VISKey":  "../data/wwwivi.key.pem",
		"Adapters":[
			{
				"Plugin":"testadapter",
				"Params": {
					"Data" : {
						"Attribute.Vehicle.VehicleIdentification.VIN":    {"Value": "TestVIN", "Public": true,"ReadOnly": true},
						"Attribute.Vehicle.UserIdentification.Users":     {"Value": ["User1", "Provider1"], "Public": true},
		
						"Signal.Drivetrain.InternalCombustionEngine.RPM": {"Value": 1000, "ReadOnly": true},
			
						"Signal.Body.Trunk.IsLocked":                     {"Value": false},
						"Signal.Body.Trunk.IsOpen":                       {"Value": true},
			
						"Signal.Cabin.Door.Row1.Right.IsLocked":          {"Value": true},
						"Signal.Cabin.Door.Row1.Right.Window.Position":   {"Value": 50},
						"Signal.Cabin.Door.Row1.Left.IsLocked":           {"Value": true},
						"Signal.Cabin.Door.Row1.Left.Window.Position":    {"Value": 23},
						"Signal.Cabin.Door.Row2.Right.IsLocked":          {"Value": false},
						"Signal.Cabin.Door.Row2.Right.Window.Position":   {"Value": 100},
						"Signal.Cabin.Door.Row2.Left.IsLocked":           {"Value": true},
						"Signal.Cabin.Door.Row2.Left.Window.Position":    {"Value": 0}
					}
				}
			}
		]
	}`

	var cfg config.Config

	decoder := json.NewDecoder(strings.NewReader(configJSON))
	// Parse config
	if err := decoder.Decode(&cfg); err != nil {
		log.Fatalf("Can't parse config: %s", err)
	}

	url, err := url.Parse(serverURL)
	if err != nil {
		log.Fatalf("Can't parse url: %s", err)
	}

	cfg.ServerURL = url.Host

	dataprovider.RegisterPlugin("testadapter", func(configJSON json.RawMessage) (adapter dataprovider.DataAdapter, err error) {
		baseAdapter, err := dataprovider.NewBaseAdapter()
		if err != nil {
			return nil, aoserrors.Wrap(err)
		}

		var sensors struct {
			Data map[string]*dataprovider.BaseData
		}

		decoder := json.NewDecoder(bytes.NewReader(configJSON))
		decoder.UseNumber()
		if err = decoder.Decode(&sensors); err != nil {
			return nil, aoserrors.Wrap(err)
		}

		baseAdapter.Data = sensors.Data

		return baseAdapter, nil
	})

	server, err := visserver.New(&cfg, &permissionProvider)
	if err != nil {
		log.Fatalf("Can't create ws server: %s", err)
	}

	// There is raise condition: after new listen is not started yet
	// so we need this delay to wait for listen
	time.Sleep(time.Second)

	ret := m.Run()

	server.Close()

	os.Exit(ret)
}

func TestGetNoAuth(t *testing.T) {
	client, err := wsclient.New("TestClient", wsclient.ClientParam{CaCertFile: caCert}, nil)
	if err != nil {
		t.Fatalf("Can't create client: %s", err)
	}
	defer client.Close()

	if err = client.Connect(serverURL); err != nil {
		t.Fatalf("Can't connect to server: %s", err)
	}

	getRequest := visprotocol.GetRequest{
		MessageHeader: visprotocol.MessageHeader{
			Action:    visprotocol.ActionGet,
			RequestID: "8765",
		},
		Path: "Attribute.Vehicle.VehicleIdentification.VIN",
	}
	getResponse := visprotocol.GetResponse{}

	if err = client.SendRequest("RequestID", getRequest.RequestID, &getRequest, &getResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if getResponse.Error != nil {
		t.Fatalf("Get request error: %s", getResponse.Error.Message)
	}
}

func TestGet(t *testing.T) {
	client, err := wsclient.New("TestClient", wsclient.ClientParam{CaCertFile: caCert}, nil)
	if err != nil {
		t.Fatalf("Can't create client: %s", err)
	}
	defer client.Close()

	if err = client.Connect(serverURL); err != nil {
		t.Fatalf("Can't connect to server: %s", err)
	}

	getRequest := visprotocol.GetRequest{
		MessageHeader: visprotocol.MessageHeader{
			Action:    visprotocol.ActionGet,
			RequestID: "8755",
		},
		Path: "Signal.Drivetrain.InternalCombustionEngine.RPM",
	}
	getResponse := visprotocol.GetResponse{}

	if err = client.SendRequest("RequestID", getRequest.RequestID, &getRequest, &getResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if getResponse.Error == nil || getResponse.Error.Number != 401 {
		t.Fatalf("Should be error 401")
	}

	authRequest := visprotocol.AuthRequest{
		MessageHeader: visprotocol.MessageHeader{
			Action:    visprotocol.ActionAuth,
			RequestID: "12345",
		},
		Tokens: visprotocol.Tokens{
			Authorization: "appUID",
		},
	}
	authResponse := visprotocol.AuthResponse{}

	if err = client.SendRequest("RequestID", authRequest.RequestID, &authRequest, &authResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if authResponse.Error != nil {
		t.Fatalf("Auth request error: %s", authResponse.Error.Message)
	}

	if err = client.SendRequest("RequestID", getRequest.RequestID, &getRequest, &getResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if authResponse.Error != nil {
		t.Fatalf("Get request error: %s", authResponse.Error.Message)
	}
}

func TestSet(t *testing.T) {
	client, err := wsclient.New("TestClient", wsclient.ClientParam{CaCertFile: caCert}, nil)
	if err != nil {
		t.Fatalf("Can't create client: %s", err)
	}
	defer client.Close()

	if err = client.Connect(serverURL); err != nil {
		t.Fatalf("Can't connect to server: %s", err)
	}

	authRequest := visprotocol.AuthRequest{
		MessageHeader: visprotocol.MessageHeader{
			Action:    visprotocol.ActionAuth,
			RequestID: "12345",
		},
		Tokens: visprotocol.Tokens{
			Authorization: "appUID",
		},
	}
	authResponse := visprotocol.AuthResponse{}

	if err = client.SendRequest("RequestID", authRequest.RequestID, &authRequest, &authResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if authResponse.Error != nil {
		t.Fatalf("Auth request error: %s", authResponse.Error.Message)
	}

	setRequest := visprotocol.SetRequest{
		MessageHeader: visprotocol.MessageHeader{
			Action:    visprotocol.ActionSet,
			RequestID: "8888",
		},
		Path: "Signal.Cabin.Door.Row1.*",
		Value: []interface{}{
			map[string]interface{}{"Right.IsLocked": true},
			map[string]interface{}{"Right.Window.Position": 100},
			map[string]interface{}{"Left.IsLocked": true},
			map[string]interface{}{"Left.Window.Position": 250},
		},
	}
	setResponse := visprotocol.GetResponse{}

	if err = client.SendRequest("RequestID", setRequest.RequestID, &setRequest, &setResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if setResponse.Error != nil {
		t.Fatalf("Set request error: %s", setResponse.Error.Message)
	}
}

func TestSubscribeUnsubscribe(t *testing.T) {
	notificationChannel := make(chan visprotocol.SubscriptionNotification, 1)

	client, err := wsclient.New("TestClient", wsclient.ClientParam{CaCertFile: caCert}, func(data []byte) {
		var notification visprotocol.SubscriptionNotification

		if err := json.Unmarshal(data, &notification); err != nil {
			t.Fatalf("Error parsing notification: %s", err)
		}

		notificationChannel <- notification
	})
	if err != nil {
		t.Fatalf("Can't create client: %s", err)
	}
	defer client.Close()

	if err = client.Connect(serverURL); err != nil {
		t.Fatalf("Can't connect to server: %s", err)
	}

	authRequest := visprotocol.AuthRequest{
		MessageHeader: visprotocol.MessageHeader{
			Action:    visprotocol.ActionAuth,
			RequestID: "12345",
		},
		Tokens: visprotocol.Tokens{
			Authorization: "appUID",
		},
	}
	authResponse := visprotocol.AuthResponse{}

	if err = client.SendRequest("RequestID", authRequest.RequestID, &authRequest, &authResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if authResponse.Error != nil {
		t.Fatalf("Auth request error: %s", authResponse.Error.Message)
	}

	// Subscribe

	subscribeRequest := visprotocol.SubscribeRequest{
		MessageHeader: visprotocol.MessageHeader{
			Action:    visprotocol.ActionSubscribe,
			RequestID: "1004",
		},
		Path: "Signal.Cabin.Door.Row1.Right.Window.Position",
	}
	subscribeResponse := visprotocol.SubscribeResponse{}

	if err = client.SendRequest("RequestID", subscribeRequest.RequestID, &subscribeRequest, &subscribeResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if subscribeResponse.Error != nil {
		t.Fatalf("Subscribe request error: %s", authResponse.Error.Message)
	}

	if subscribeResponse.SubscriptionID == "" {
		t.Fatalf("No subscriptionId")
	}

	subscriptionID := subscribeResponse.SubscriptionID

	// Change data

	setRequest := visprotocol.SetRequest{
		MessageHeader: visprotocol.MessageHeader{
			Action:    visprotocol.ActionSet,
			RequestID: "1004",
		},
		Path:  "Signal.Cabin.Door.Row1.Right.Window.Position",
		Value: 123,
	}
	setResponse := visprotocol.GetResponse{}

	if err = client.SendRequest("RequestID", setRequest.RequestID, &setRequest, &setResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if setResponse.Error != nil {
		t.Fatalf("Set request error: %s", setResponse.Error.Message)
	}

	// Wait for notification

	select {
	case notification := <-notificationChannel:
		if notification.Action != "subscription" || notification.SubscriptionID != subscriptionID || notification.Value.(float64) != 123.0 {
			t.Fatalf("Unexpected value")
		}

		if notification.Error != nil {
			t.Fatalf("Unexpected error for subscription: %s", notification.Error.Message)
		}

	case <-time.After(1 * time.Second):
		t.Fatal("Waiting for subscription notification timeout")
	}

	// Unsubscribe wrong id

	unsubscribeRequest := visprotocol.UnsubscribeRequest{
		MessageHeader: visprotocol.MessageHeader{
			Action:    visprotocol.ActionUnsubscribe,
			RequestID: "1004",
		},
		SubscriptionID: "1",
	}
	unsubscribeResponse := visprotocol.UnsubscribeResponse{}

	if err = client.SendRequest("RequestID", unsubscribeRequest.RequestID, &unsubscribeRequest, &unsubscribeResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if unsubscribeResponse.Error == nil {
		t.Fatal("Unexpected positive response")
	}

	// Unsubscribe

	unsubscribeRequest.SubscriptionID = subscriptionID
	unsubscribeResponse = visprotocol.UnsubscribeResponse{}

	if err = client.SendRequest("RequestID", unsubscribeRequest.RequestID, &unsubscribeRequest, &unsubscribeResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if unsubscribeResponse.Error != nil {
		t.Fatalf("Unsubscribe request error: %s", unsubscribeResponse.Error.Message)
	}

	// UnsubscribeAll

	unsubscribeAllRequest := visprotocol.UnsubscribeAllRequest{
		MessageHeader: visprotocol.MessageHeader{
			Action: visprotocol.ActionUnsubscribeAll,
		},
	}
	unsubscribeAllResponse := visprotocol.UnsubscribeAllResponse{}

	if err = client.SendRequest("RequestID", unsubscribeAllRequest.RequestID, &unsubscribeAllRequest, &unsubscribeAllResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if unsubscribeAllResponse.Error != nil {
		t.Fatalf("Unsubscribe all request error: %s", setResponse.Error.Message)
	}
}
