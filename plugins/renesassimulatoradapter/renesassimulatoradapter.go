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

package renesassimulatoradapter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"github.com/aoscloud/aos_common/aoserrors"
	"github.com/aoscloud/aos_vis/dataprovider"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// RenesasSimulatorAdapter Renesas simulator adapter.
type RenesasSimulatorAdapter struct {
	httpServer  *http.Server
	upgrader    websocket.Upgrader
	baseAdapter *dataprovider.BaseAdapter
	signalMap   map[string]string
}

type config struct {
	ServerURL string
	SignalMap map[string]string `json:"Signals"`
}

type simulatorMessage struct {
	Command  string      `json:"cmd"`
	Argument interface{} `json:"arg"`
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// New creates adapter instance.
func New(configJSON json.RawMessage) (adapter dataprovider.DataAdapter, err error) {
	log.Info("Create Renesas simulator adapter")

	localAdapter := new(RenesasSimulatorAdapter)

	var config config

	// Parse config
	err = json.Unmarshal(configJSON, &config)
	if err != nil {
		return nil, aoserrors.Wrap(err)
	}

	localAdapter.signalMap = config.SignalMap

	if localAdapter.baseAdapter, err = dataprovider.NewBaseAdapter(); err != nil {
		return nil, aoserrors.Wrap(err)
	}

	localAdapter.baseAdapter.Name = "RenesasSimulatorAdapter"

	for _, signal := range localAdapter.signalMap {
		if signal != "" {
			localAdapter.baseAdapter.Data[signal] = &dataprovider.BaseData{}
		}
	}

	localAdapter.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", localAdapter.handleConnection)

	localAdapter.httpServer = &http.Server{Addr: config.ServerURL, Handler: serveMux}

	go func() {
		log.WithField("address", config.ServerURL).Debug("Listen for Renesas simulator")

		if err := localAdapter.httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Error("Server listening error: ", err)
			return
		}
	}()

	return localAdapter, nil
}

// Close closes adapter.
func (adapter *RenesasSimulatorAdapter) Close() {
	log.Info("Close Renesas simulator adapter")

	_ = adapter.httpServer.Shutdown(context.Background())
	adapter.baseAdapter.Close()
}

// GetName returns adapter name.
func (adapter *RenesasSimulatorAdapter) GetName() (name string) {
	return adapter.baseAdapter.GetName()
}

// GetPathList returns list of all pathes for this adapter.
func (adapter *RenesasSimulatorAdapter) GetPathList() (pathList []string, err error) {
	pathList, err = adapter.baseAdapter.GetPathList()
	if err != nil {
		return pathList, aoserrors.Wrap(err)
	}

	return pathList, nil
}

// IsPathPublic returns true if requested data accessible without authorization.
func (adapter *RenesasSimulatorAdapter) IsPathPublic(path string) (result bool, err error) {
	adapter.baseAdapter.Lock()
	defer adapter.baseAdapter.Unlock()

	// TODO: return false, once authorization is integrated

	return true, nil
}

// GetData returns data by path.
func (adapter *RenesasSimulatorAdapter) GetData(pathList []string) (data map[string]interface{}, err error) {
	data, err = adapter.baseAdapter.GetData(pathList)
	if err != nil {
		return data, aoserrors.Wrap(err)
	}

	return data, nil
}

// SetData sets data by pathes.
func (adapter *RenesasSimulatorAdapter) SetData(data map[string]interface{}) (err error) {
	return aoserrors.New("operation is not supported")
}

// GetSubscribeChannel returns channel on which data changes will be sent.
func (adapter *RenesasSimulatorAdapter) GetSubscribeChannel() (channel <-chan map[string]interface{}) {
	return adapter.baseAdapter.SubscribeChannel
}

// Subscribe subscribes for data changes.
func (adapter *RenesasSimulatorAdapter) Subscribe(pathList []string) (err error) {
	return aoserrors.Wrap(adapter.baseAdapter.Subscribe(pathList))
}

// Unsubscribe unsubscribes from data changes.
func (adapter *RenesasSimulatorAdapter) Unsubscribe(pathList []string) (err error) {
	return aoserrors.Wrap(adapter.baseAdapter.Unsubscribe(pathList))
}

// UnsubscribeAll unsubscribes from all data changes.
func (adapter *RenesasSimulatorAdapter) UnsubscribeAll() (err error) {
	return aoserrors.Wrap(adapter.baseAdapter.UnsubscribeAll())
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func (adapter *RenesasSimulatorAdapter) handleConnection(w http.ResponseWriter, r *http.Request) {
	log.WithField("RemoteAddr", r.RemoteAddr).Debug("Renesas simulator connection request")

	if !websocket.IsWebSocketUpgrade(r) {
		log.Error("New connection is not websocket")
		return
	}

	connection, err := adapter.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("Can't make websocket connection: ", err)
		return
	}

	for {
		messageType, message, err := connection.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure) &&
				!strings.Contains(err.Error(), "use of closed network connection") {
				log.Errorf("Error reading socket: %s", err)
			}

			break
		}

		if messageType == websocket.TextMessage {
			log.Debugf("Renesas adapter receive: %s", string(message))

			var simulatorMessage simulatorMessage

			if err := json.Unmarshal(message, &simulatorMessage); err != nil {
				log.Errorf("Can't parse message: %s", err)
				continue
			}

			switch simulatorMessage.Command {
			case "data":
				result := make(map[string]interface{})

				if err := adapter.handleSimulatorData("", simulatorMessage.Argument, result); err != nil {
					log.Errorf("Can't parse simulator data: %s", err)
				}

				// Multiply longitude by -1, fix for Renesas simulator
				longitude, ok := result["Signal.Cabin.Infotainment.Navigation.CurrentLocation.Longitude"]
				if ok {
					floatLongitude, ok := longitude.(float64)
					if ok {
						result["Signal.Cabin.Infotainment.Navigation.CurrentLocation.Longitude"] = -1 * floatLongitude
					}
				}

				if len(result) != 0 {
					if err = adapter.baseAdapter.SetData(result); err != nil {
						log.Errorf("Can't set data to adapter: %s", err)
					}
				}

			default:
				log.WithField("command", simulatorMessage.Command).Warning("Unsupported command received")
			}
		} else {
			log.WithField("format", messageType).Warning("Incoming message in unsupported format")
		}
	}
}

func (adapter *RenesasSimulatorAdapter) handleSimulatorData(prefix string, data interface{},
	result map[string]interface{}) (err error) {
	if data == nil {
		log.Error("Nil data received")
		return nil
	}

	keyMap, ok := data.(map[string]interface{})
	if !ok {
		signal, ok := adapter.signalMap[prefix]
		if !ok {
			log.WithFields(log.Fields{"key": prefix, "value": data}).Warn("Unsupported signal received")
			return nil
		}

		if signal != "" {
			result[signal] = data
		}

		return nil
	}

	if prefix != "" {
		prefix += "."
	}

	for key, value := range keyMap {
		if err = adapter.handleSimulatorData(prefix+key, value, result); err != nil {
			return aoserrors.Wrap(err)
		}
	}

	return nil
}
