package wsserver

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"gitpct.epam.com/epmd-aepr/aos_vis/vehicledataprovider"
)

/*******************************************************************************
 * Consts
 ******************************************************************************/

const (
	actionGet = "get"
)

const (
	getPermission = iota
	setPermission
)

/*******************************************************************************
 * Types
 ******************************************************************************/
type WsClientConnection struct {
	name         string
	wsConn       *websocket.Conn
	isAuthorised bool
}

type requestType struct {
	Action string `json:"action"`
}

type requestGet struct {
	Action    string `json:"action"`
	Path      string `json:"path"`
	RequestId string `json:"requestId"`
}

type getSuccessResponse struct {
	Action    string      `json:"action"`
	RequestId string      `json:"requestId"`
	Value     interface{} `json:"value"`
	Timestamp int64       `json:"timestamp"`
}

type errorResponce struct {
	Action    string    `json:"action"`
	RequestId string    `json:"requestId"`
	Error     errorInfo `json:"error"`
	Timestamp int64     `json:"timestamp"`
}

//TODO: add map arror number message
type errorInfo struct {
	Number  int
	Reason  string
	Message string
}

/*******************************************************************************
 * Public
 ******************************************************************************/
//create web socket client connection information
func NewClientConn(wsConn *websocket.Conn) (wsClientCon *WsClientConnection, err error) {
	log.Debug("NewClientConn")
	if wsConn == nil {
		return wsClientCon, errors.New("Null connection")
	}
	var localConnection WsClientConnection
	localConnection.wsConn = wsConn
	wsClientCon = &localConnection

	return wsClientCon, nil
}

//ProcessConnection process incommming websoket connection
func (client *WsClientConnection) ProcessConnection() {
	//TODO: add select fro processing subscription channel
	log.Debug("Start processing new WS connection")
	for {
		mt, message, err := client.wsConn.ReadMessage()
		if err != nil {
			log.Warning("Can't read from WS: ", err)
			break
		}
		if mt == 1 {
			client.processImcommingMessage(message)
		}

		log.Debug("message type: ", mt)

		//client.WriteMessage(message)

	}
	log.Debug("Stop processConnection")
}

func (client *WsClientConnection) WriteMessage(data []byte) {
	err := client.wsConn.WriteMessage(1, data)
	if err != nil {
		log.Error("Can't write to WS: ", err)
	}
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func (client *WsClientConnection) processImcommingMessage(data []byte) {
	log.Info("recv: ", string(data))

	var rType requestType

	err := json.Unmarshal(data, &rType)
	if err != nil {
		log.Error("Error get actionType  ", err)
		return
	}
	log.Info("action: ", rType.Action)
	switch string(rType.Action) {
	case actionGet:
		var rGet requestGet
		err := json.Unmarshal(data, &rGet)
		if err != nil {
			log.Error("Error parce Get request  ", err)
			msg := errorResponce{Action: rType.Action, Timestamp: time.Now().Unix(), Error: errorInfo{Number: 400}}
			respJson, err := json.Marshal(msg)
			if err != nil {
				log.Warn("Error marshall json: ", err)
				return
			}
			client.WriteMessage(respJson)
		}
		responce := client.processGetRequest(&rGet)
		client.WriteMessage(responce)
	default:
		msg := errorResponce{Action: rType.Action, Timestamp: time.Now().Unix(), Error: errorInfo{Number: 400}}
		respJson, err := json.Marshal(msg)
		if err != nil {
			log.Warn("Error marshall json: ", err)
			return
		}
		client.WriteMessage(respJson)
	}
}

func (client *WsClientConnection) processGetRequest(request *requestGet) (resp []byte) {
	var err error
	var msg interface{}
	needToAskForData := false
	if vehicledataprovider.IsPublicPath(request.Path) == true {
		needToAskForData = true
	} else {
		if client.isAuthorised == false {
			log.Info("Client not Authorised send responce 403")
			msg = errorResponce{Action: actionGet, RequestId: request.RequestId, Error: errorInfo{Number: 403}, Timestamp: time.Now().Unix()}
		} else {
			if client.checkPermission(request.Path, getPermission) == true {
				needToAskForData = true
			} else {
				msg = errorResponce{Action: actionGet, RequestId: request.RequestId,
					Error:     errorInfo{Number: 403, Message: "The user is not permitted to access the requested resource"},
					Timestamp: time.Now().Unix()}
			}
		}
	}

	if needToAskForData == true {
		vehData, err := vehicledataprovider.GetDataByPath(request.Path)
		if err != nil {
			log.Debug("No data for path ", request.Path)
			msg = errorResponce{Action: actionGet, RequestId: request.RequestId, Error: errorInfo{Number: 404}, Timestamp: time.Now().Unix()}

		} else {
			log.Debug("data from dataprovider ", vehData)
			msg = getSuccessResponse{Action: actionGet, RequestId: request.RequestId, Value: vehData, Timestamp: time.Now().Unix()}
		}
	}

	resp, err = json.Marshal(msg)
	if err != nil {
		log.Warn("Error marshall json: ", err)
		return []byte("Error marshall json")
	}
	return resp
}

//check permission got set or get
func (client *WsClientConnection) checkPermission(path string, permission uint) (resp bool) {
	return true
}
