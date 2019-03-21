package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"gitpct.epam.com/epmd-aepr/aos_vis/dataadapter"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// RenesasSimulatorAdapter Renesas simulator adapter
type RenesasSimulatorAdapter struct {
	httpServer  *http.Server
	upgrader    websocket.Upgrader
	baseAdapter *dataadapter.BaseAdapter
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

// NewAdapter creates adapter instance
func NewAdapter(configJSON []byte) (adapter dataadapter.DataAdapter, err error) {
	log.Info("Create Renesas simulator adapter")

	localAdapter := new(RenesasSimulatorAdapter)

	var config config

	// Parse config
	err = json.Unmarshal(configJSON, &config)
	if err != nil {
		return nil, err
	}

	localAdapter.signalMap = config.SignalMap

	if localAdapter.baseAdapter, err = dataadapter.NewBaseAdapter(); err != nil {
		return nil, err
	}

	localAdapter.baseAdapter.Name = "RenesasSimulatorAdapter"

	for _, signal := range localAdapter.signalMap {
		if signal != "" {
			localAdapter.baseAdapter.Data[signal] = &dataadapter.BaseData{}
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

		if err := localAdapter.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Error("Server listening error: ", err)
			return
		}
	}()

	return localAdapter, nil
}

// Close closes adapter
func (adapter *RenesasSimulatorAdapter) Close() {
	log.Info("Close Renesas simulator adapter")

	adapter.httpServer.Shutdown(context.Background())
	adapter.baseAdapter.Close()
}

// GetName returns adapter name
func (adapter *RenesasSimulatorAdapter) GetName() (name string) {
	return adapter.baseAdapter.GetName()
}

// GetPathList returns list of all pathes for this adapter
func (adapter *RenesasSimulatorAdapter) GetPathList() (pathList []string, err error) {
	return adapter.baseAdapter.GetPathList()
}

// IsPathPublic returns true if requested data accessible without authorization
func (adapter *RenesasSimulatorAdapter) IsPathPublic(path string) (result bool, err error) {
	adapter.baseAdapter.Lock()
	defer adapter.baseAdapter.Unlock()

	// TODO: return false, once authorization is integrated

	return true, nil
}

// GetData returns data by path
func (adapter *RenesasSimulatorAdapter) GetData(pathList []string) (data map[string]interface{}, err error) {
	return adapter.baseAdapter.GetData(pathList)
}

// SetData sets data by pathes
func (adapter *RenesasSimulatorAdapter) SetData(data map[string]interface{}) (err error) {
	return errors.New("operation is not supported")
}

// GetSubscribeChannel returns channel on which data changes will be sent
func (adapter *RenesasSimulatorAdapter) GetSubscribeChannel() (channel <-chan map[string]interface{}) {
	return adapter.baseAdapter.SubscribeChannel
}

// Subscribe subscribes for data changes
func (adapter *RenesasSimulatorAdapter) Subscribe(pathList []string) (err error) {
	return adapter.baseAdapter.Subscribe(pathList)
}

// Unsubscribe unsubscribes from data changes
func (adapter *RenesasSimulatorAdapter) Unsubscribe(pathList []string) (err error) {
	return adapter.baseAdapter.Unsubscribe(pathList)
}

// UnsubscribeAll unsubscribes from all data changes
func (adapter *RenesasSimulatorAdapter) UnsubscribeAll() (err error) {
	return adapter.baseAdapter.UnsubscribeAll()
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func (adapter *RenesasSimulatorAdapter) handleConnection(w http.ResponseWriter, r *http.Request) {
	log.WithField("RemoteAddr", r.RemoteAddr).Debug("Renesas simulator connection request")

	if websocket.IsWebSocketUpgrade(r) != true {
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
			return err
		}
	}
	return nil
}
