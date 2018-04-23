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
	actionGet  = "get"
	actionAuth = "authorize"
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
	permisssions permissionData
}

type requestType struct {
	Action string `json:"action"`
}

type requestGet struct {
	Action    string `json:"action"`
	Path      string `json:"path"`
	RequestId string `json:"requestId"`
}

type requestAuth struct {
	Action    string        `json:"action"`
	Tokens    tockensStruct `json:"tokens"`
	RequestId string        `json:"requestId"`
}

type tockensStruct struct {
	Authorization      *string `json:"authorization"`
	www_vehicle_device *string `json:"www-vehicle-device"`
}

type getSuccessResponse struct {
	Action    string      `json:"action"`
	RequestId string      `json:"requestId"`
	Value     interface{} `json:"value"`
	Timestamp int64       `json:"timestamp"`
}

type authSuccessResponse struct {
	Action    string `json:"action"`
	RequestId string `json:"requestId"`
	Ttl       int64  `json:"TTL"`
	Timestamp int64  `json:"timestamp"`
}

type errorResponse struct {
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

type permissionData map[string]string

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
			client.processIncommingMessage(message)
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

func (client *WsClientConnection) processIncommingMessage(data []byte) {
	log.Info("receive : ", string(data))

	var rType requestType

	err := json.Unmarshal(data, &rType)
	if err != nil {
		log.Error("Error get actionType  ", err)
		client.senderrorResponse("", "", &errorInfo{Number: 400})
		return
	}
	log.Info("action: ", rType.Action)

	switch string(rType.Action) {
	case actionGet:
		var rGet requestGet
		err := json.Unmarshal(data, &rGet)
		if err != nil {
			log.Error("Error parse Get request  ", err)
			client.senderrorResponse("", "", &errorInfo{Number: 400})
			return
		}
		responce := client.processGetRequest(&rGet)
		client.WriteMessage(responce)

	case actionAuth:
		var rAuth requestAuth
		err := json.Unmarshal(data, &rAuth)
		if err != nil {
			log.Error("Error parse Auth request  ", err)
			client.senderrorResponse(actionAuth, "", &errorInfo{Number: 400})
			return
		}

		if rAuth.Tokens.Authorization == nil {
			log.Error("Error Tokens.Authorization = nil  ")
			client.senderrorResponse(actionAuth, rAuth.RequestId, &errorInfo{Number: 400})
			return
		}
		responce := client.processAuthRequest(&rAuth)
		client.WriteMessage(responce)

	default:
		log.Error("unsupported action type = ", rType.Action)
		client.senderrorResponse(rType.Action, "", &errorInfo{Number: 400})
	}
}

// process Get request
func (client *WsClientConnection) processGetRequest(request *requestGet) (resp []byte) {
	var err error
	var msg interface{}
	needToAskForData := false
	if vehicledataprovider.IsPublicPath(request.Path) == true {
		needToAskForData = true
	} else {
		if client.isAuthorised == false {
			log.Info("Client not Authorized send response 403")
			msg = errorResponse{Action: actionGet, RequestId: request.RequestId, Error: errorInfo{Number: 403}, Timestamp: time.Now().Unix()}
		} else {
			if client.checkPermission(request.Path, getPermission) == true {
				needToAskForData = true
			} else {
				msg = errorResponse{Action: actionGet, RequestId: request.RequestId,
					Error:     errorInfo{Number: 403, Message: "The user is not permitted to access the requested resource"},
					Timestamp: time.Now().Unix()}
			}
		}
	}

	if needToAskForData == true {
		vehData, err := vehicledataprovider.GetDataByPath(request.Path)
		if err != nil {
			log.Warn("No data for path ", request.Path)
			msg = errorResponse{Action: actionGet, RequestId: request.RequestId, Error: errorInfo{Number: 404}, Timestamp: time.Now().Unix()}

		} else {
			log.Debug("Data from dataprovider ", vehData)
			msg = getSuccessResponse{Action: actionGet, RequestId: request.RequestId, Value: vehData, Timestamp: time.Now().Unix()}
		}
	}

	resp, err = json.Marshal(msg)
	if err != nil {
		log.Warn("Error marshal json: ", err)
		return []byte("Error marshal json")
	}
	return resp
}

func (client *WsClientConnection) processAuthRequest(request *requestAuth) (resp []byte) {
	var msg interface{}
	var err error

	if client.isAuthorised == false {
		//TODO: add retry count
		data, errInfo := getPermissionListForClient(*request.Tokens.Authorization)
		if errInfo != nil {
			log.Error("Error auth code ", errInfo.Number)
			msg = errorResponse{Action: actionAuth, RequestId: request.RequestId, Error: *errInfo, Timestamp: time.Now().Unix()}
		} else {
			client.permisssions = data
			client.isAuthorised = true
			msg = authSuccessResponse{Action: actionAuth, RequestId: request.RequestId, Ttl: 10000, Timestamp: time.Now().Unix()}
		}
	} else {
		log.Info("Token ", request.Tokens.Authorization, " already authorised")
		msg = authSuccessResponse{Action: actionAuth, RequestId: request.RequestId, Ttl: 10000, Timestamp: time.Now().Unix()}
	}

	resp, err = json.Marshal(msg)
	if err != nil {
		log.Warn("Error marshal json: ", err)
		return []byte("Error marshal json")
	}
	return resp
}

//check permission got set or get
func (client *WsClientConnection) checkPermission(path string, permission uint) (resp bool) {
	return true
}

func (client *WsClientConnection) senderrorResponse(action string, reqID string, errResp *errorInfo) {
	msg := errorResponse{Action: action, Timestamp: time.Now().Unix(), Error: *errResp, RequestId: reqID}
	respJson, err := json.Marshal(msg)
	if err != nil {
		log.Warn("Error marshal json: ", err)
		return
	}
	client.WriteMessage(respJson)
}

func getPermissionListForClient(token string) (data permissionData, errInfo *errorInfo) {
	//TODO imlements d-bus call get
	data = make(permissionData)
	log.Info("get Permission List For Client token ", token)
	data["Signal.Drivetrain.InternalCombustionEngine.RPM"] = "r"
	return data, nil
}
