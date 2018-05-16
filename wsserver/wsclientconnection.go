package wsserver

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"gitpct.epam.com/epmd-aepr/aos_vis/vehicledataprovider"
)

/*******************************************************************************
 * Consts
 ******************************************************************************/

const (
	actionGet            = "get"
	actionAuth           = "authorize"
	actionSubscribe      = "subscribe"
	actionUnsubscribe    = "unsubscribe"
	actionUnsubscribeAll = "unsubscribeAll"
)

const (
	getPermission = iota
	setPermission
)

/*******************************************************************************
 * Types
 ******************************************************************************/
type WsClientConnection struct {
	name           string
	wsConn         *websocket.Conn
	isAuthorised   bool
	permissions    permissionData
	subscriptionCh chan interface{} //TODO: change to struct from dataprovider
}

type requestType struct {
	Action    string `json:"action"`
	RequestId string `json:"requestId"`
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

type requestSubscribe struct {
	Action    string  `json:"action"`
	Path      string  `json:"path"`
	Filters   *string `json:"filters"` //TODO: will be implemented later
	RequestId string  `json:"requestId"`
}

type requestUnsubscribe struct {
	Action         string `json:"action"`
	SubscriptionId string `json:"subscriptionId"`
	RequestId      string `json:"requestId"`
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

type subscribeUnsubscribeSuccessResponse struct {
	Action         string `json:"action"`
	SubscriptionId string `json:"subscriptionId"`
	RequestId      string `json:"requestId"`
	Timestamp      int64  `json:"timestamp"`
}

type unsubscribeAllSuccessResponse struct {
	Action    string `json:"action"`
	RequestId string `json:"requestId"`
	Timestamp int64  `json:"timestamp"`
}

type errorResponse struct {
	Action         string    `json:"action"`
	RequestId      string    `json:"requestId"`
	Error          errorInfo `json:"error"`
	SubscriptionId *string   `json:"subscriptionId"`
	Timestamp      int64     `json:"timestamp"`
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

	wsClientCon.subscriptionCh = make(chan interface{}, 100)

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
			client.senderrorResponse(actionAuth, rType.RequestId, &errorInfo{Number: 400})
			return
		}

		if rAuth.Tokens.Authorization == nil {
			log.Error("Error Tokens.Authorization = nil  ")
			client.senderrorResponse(actionAuth, rAuth.RequestId, &errorInfo{Number: 400})
			return
		}
		responce := client.processAuthRequest(&rAuth)
		client.WriteMessage(responce)

	case actionSubscribe:
		var rSubs requestSubscribe
		err := json.Unmarshal(data, &rSubs)
		if err != nil {
			log.Error("Error parse Subscribe request  ", err)
			client.senderrorResponse(actionSubscribe, rType.RequestId, &errorInfo{Number: 400})
			return
		}
		if rSubs.Filters != nil {
			log.Warn("Filter currently not implemented. Filters will be ignored")
		}
		log.Debug("req subs", rSubs)
		responce := client.processSubscibeRequest(&rSubs)
		client.WriteMessage(responce)

	case actionUnsubscribe:
		var rUnsubs requestUnsubscribe
		err := json.Unmarshal(data, &rUnsubs)
		if err != nil {
			log.Error("Error parse Unsubscribe request  ", err)
			client.senderrorResponse(actionUnsubscribe, rType.RequestId, &errorInfo{Number: 400})
			return
		}
		log.Debug("req Unsubs", rUnsubs)
		responce := client.processUnsubscibeRequest(&rUnsubs)
		client.WriteMessage(responce)

	case actionUnsubscribeAll:
		log.Debug("req UnsubscribeAll")
		err := vehicledataprovider.RegestrateUnSubscribAll(client.subscriptionCh)
		if err != nil {
			client.senderrorResponse(actionUnsubscribeAll, rType.RequestId, &errorInfo{Number: 400})
			return
		}
		msg := unsubscribeAllSuccessResponse{Action: actionUnsubscribeAll, RequestId: rType.RequestId, Timestamp: time.Now().Unix()}

		var resp []byte
		resp, err = json.Marshal(msg)
		if err != nil {
			log.Warn("Error marshal json: ", err)
			resp = []byte("Error marshal json")
		}
		client.WriteMessage(resp)

	default:
		log.Error("unsupported action type = ", rType.Action)
		client.senderrorResponse(rType.Action, rType.RequestId, &errorInfo{Number: 400})
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

// process Auth request
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
			client.permissions = data
			client.isAuthorised = true
			msg = authSuccessResponse{Action: actionAuth, RequestId: request.RequestId, Ttl: 10000, Timestamp: time.Now().Unix()}
		}
	} else {
		log.Info("Token ", request.Tokens.Authorization, " already authorized")
		msg = authSuccessResponse{Action: actionAuth, RequestId: request.RequestId, Ttl: 10000, Timestamp: time.Now().Unix()}
	}

	resp, err = json.Marshal(msg)
	if err != nil {
		log.Warn("Error marshal json: ", err)
		return []byte("Error marshal json")
	}
	return resp
}

// process Subscibe request
func (client *WsClientConnection) processSubscibeRequest(request *requestSubscribe) (resp []byte) {
	var err error
	var msg interface{}
	isPermissionOK := false
	if vehicledataprovider.IsPublicPath(request.Path) == true {
		isPermissionOK = true
	} else {
		if client.isAuthorised == false {
			log.Info("Client not Authorized send response 403 id", request.RequestId)
			msg = errorResponse{Action: actionSubscribe, RequestId: request.RequestId, Error: errorInfo{Number: 403}, Timestamp: time.Now().Unix()}
		} else {
			if client.checkPermission(request.Path, getPermission) == true {
				isPermissionOK = true
			} else {
				msg = errorResponse{Action: actionSubscribe, RequestId: request.RequestId,
					Error:     errorInfo{Number: 403, Message: "The user is not permitted to access the requested resource"},
					Timestamp: time.Now().Unix()}
			}
		}
	}

	if isPermissionOK == true {
		subscrId, err := vehicledataprovider.RegestrateSubscriptionClient(client.subscriptionCh, request.Path)
		if err != nil {
			log.Warn("No data for path ", request.Path)
			msg = errorResponse{Action: actionSubscribe, RequestId: request.RequestId, Error: errorInfo{Number: 404}, Timestamp: time.Now().Unix()}

		} else {
			log.Debug("SubscriptionId from dataprovider ", subscrId)
			msg = subscribeUnsubscribeSuccessResponse{Action: actionSubscribe, RequestId: request.RequestId, SubscriptionId: subscrId, Timestamp: time.Now().Unix()}
		}
	}

	resp, err = json.Marshal(msg)
	if err != nil {
		log.Warn("Error marshal json: ", err)
		return []byte("Error marshal json")
	}
	return resp
}

// process Unsubscibe request
func (client *WsClientConnection) processUnsubscibeRequest(request *requestUnsubscribe) (resp []byte) {
	var err error
	var msg interface{}
	err = vehicledataprovider.RegestrateUnSubscription(client.subscriptionCh, request.SubscriptionId)
	if err != nil {
		log.Warn("Can' unsibscribe from ID", request.SubscriptionId)
		msg = errorResponse{Action: actionUnsubscribe, SubscriptionId: &request.SubscriptionId, RequestId: request.RequestId, Error: errorInfo{Number: 404}, Timestamp: time.Now().Unix()}
	} else {
		log.Debug("UnSubscription from ID ", request.SubscriptionId)
		msg = subscribeUnsubscribeSuccessResponse{Action: actionUnsubscribe, SubscriptionId: request.SubscriptionId, RequestId: request.RequestId, Timestamp: time.Now().Unix()}
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
	regexpStr := strings.Replace(path, ".", "[.]", -1)
	regexpStr = strings.Replace(regexpStr, "*", ".*?", -1)
	regexpStr = "^" + regexpStr
	log.Debug("filter =", regexpStr)
	var validID = regexp.MustCompile(regexpStr)

	for k, v := range client.permissions {
		if validID.MatchString(k) == true {
			switch permission {
			case getPermission:
				if v == "r" || v == "wr" || v == "rw" {
					resp = true
					log.Info("GET permission present for path ", path)
					return
				}

			case setPermission:
				if v == "w" || v == "wr" || v == "rw" {
					resp = true
					log.Info("SET permission present for path ", path)
					return
				}
			}
		}
	}
	log.Warn("Permission NOT present for path ", path)
	return false
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
	//TODO: implement d-bus call get
	data = make(permissionData)
	log.Info("get Permission List For Client token ", token)
	data["Signal.Drivetrain.InternalCombustionEngine.RPM"] = "r"
	return data, nil
}
