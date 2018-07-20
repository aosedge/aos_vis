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
	"gitpct.epam.com/epmd-aepr/aos_vis/visdbusclient"
)

/*******************************************************************************
 * Consts
 ******************************************************************************/

const (
	actionGet            = "get"
	actionSet            = "set"
	actionAuth           = "authorize"
	actionSubscribe      = "subscribe"
	actionUnsubscribe    = "unsubscribe"
	actionUnsubscribeAll = "unsubscribeAll"
	actionSubscription   = "subscription"
)

const (
	getPermission = iota
	setPermission
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// WsClientConnection websocket client connection structure
type WsClientConnection struct {
	name           string
	wsConn         *websocket.Conn
	isAuthorized   bool
	permissions    permissionData
	subscriptionCh chan vehicledataprovider.SubscriptionOutputData //TODO: change to struct from dataprovider
}

type requestType struct {
	Action    string `json:"action"`
	RequestID string `json:"requestId"`
}

type requestGet struct {
	Action    string `json:"action"`
	Path      string `json:"path"`
	RequestID string `json:"requestId"`
}

type requestSet struct {
	Action    string      `json:"action"`
	Path      string      `json:"path"`
	Value     interface{} `json:"value"`
	RequestID string      `json:"requestId"`
}

type requestAuth struct {
	Action    string       `json:"action"`
	Tokens    tokensStruct `json:"tokens"`
	RequestID string       `json:"requestId"`
}

type requestSubscribe struct {
	Action    string  `json:"action"`
	Path      string  `json:"path"`
	Filters   *string `json:"filters"` //TODO: will be implemented later
	RequestID string  `json:"requestId"`
}

type requestUnsubscribe struct {
	Action         string `json:"action"`
	SubscriptionID string `json:"subscriptionId"`
	RequestID      string `json:"requestId"`
}

type tokensStruct struct {
	Authorization    *string `json:"authorization"`
	WwwVehicleDevice *string `json:"www-vehicle-device"`
}

type getSuccessResponse struct {
	Action    string      `json:"action"`
	RequestID string      `json:"requestId"`
	Value     interface{} `json:"value"`
	Timestamp int64       `json:"timestamp"`
}

type setSuccessResponse struct {
	Action    string `json:"action"`
	RequestID string `json:"requestId"`
	Timestamp int64  `json:"timestamp"`
}

type authSuccessResponse struct {
	Action    string `json:"action"`
	RequestID string `json:"requestId"`
	TTL       int64  `json:"TTL"`
	Timestamp int64  `json:"timestamp"`
}

type subscribeUnsubscribeSuccessResponse struct {
	Action         string `json:"action"`
	SubscriptionID string `json:"subscriptionId"`
	RequestID      string `json:"requestId"`
	Timestamp      int64  `json:"timestamp"`
}

type unsubscribeAllSuccessResponse struct {
	Action    string `json:"action"`
	RequestID string `json:"requestId"`
	Timestamp int64  `json:"timestamp"`
}

type sunscribeNotification struct {
	Action         string      `json:"action"`
	SubscriptionID string      `json:"subscriptionId"`
	Value          interface{} `json:"value"`
	Timestamp      int64       `json:"timestamp"`
}

type errorResponse struct {
	Action         string    `json:"action"`
	RequestID      string    `json:"requestId"`
	Error          errorInfo `json:"error"`
	SubscriptionID *string   `json:"subscriptionId"`
	Timestamp      int64     `json:"timestamp"`
}

//TODO: add map error number message
type errorInfo struct {
	Number  int
	Reason  string
	Message string
}

type permissionData map[string]string

/*******************************************************************************
 * Public
 ******************************************************************************/

//NewClientConn create web socket client connection information
func NewClientConn(wsConn *websocket.Conn) (wsClientCon *WsClientConnection, err error) {
	log.Debug("NewClientConn")
	if wsConn == nil {
		return wsClientCon, errors.New("Null connection")
	}
	var localConnection WsClientConnection
	localConnection.wsConn = wsConn
	wsClientCon = &localConnection

	wsClientCon.subscriptionCh = make(chan vehicledataprovider.SubscriptionOutputData, 100)

	return wsClientCon, nil
}

func (client *WsClientConnection) processSubscriptionChannel() {
	for {
		data, more := <-client.subscriptionCh
		if more {
			msg := sunscribeNotification{Action: actionSubscription,
				SubscriptionID: data.ID,
				Value:          data.OutData,
				Timestamp:      getCurTime()}
			resp, err := json.Marshal(msg)
			if err != nil {
				log.Warn("Error marshal json: ", err)
				//TODO: create error subscriptionmsg
			}
			client.WriteMessage(resp)
		} else {
			log.Debug("channelClosed")
			return
		}

	}
}

//ProcessConnection process incommoding websocket connection
func (client *WsClientConnection) ProcessConnection() {
	//TODO: add select fro processing subscription channel
	log.Debug("Start processing new WS connection")
	go client.processSubscriptionChannel()
	for {
		mt, message, err := client.wsConn.ReadMessage()
		if err != nil {
			log.Warning("Can't read from WS: ", err)
			vehicledataprovider.GetInstance().RegestrateUnSubscribAll(client.subscriptionCh)
			close(client.subscriptionCh)
			break
		}
		if mt == 1 {
			client.processIncomingMessage(message)
		}
	}
	log.Debug("Stop processConnection")
}

//WriteMessage write data to opened WS connection
func (client *WsClientConnection) WriteMessage(data []byte) {
	err := client.wsConn.WriteMessage(1, data)
	if err != nil {
		log.Error("Can't write to WS: ", err)
	}
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func (client *WsClientConnection) processIncomingMessage(data []byte) {
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

	case actionSet:
		var rSet requestSet
		err := json.Unmarshal(data, &rSet)
		if err != nil {
			log.Error("Error parse Set request  ", err)
			client.senderrorResponse("", "", &errorInfo{Number: 400})
			return
		}
		responce := client.processSetRequest(&rSet)
		client.WriteMessage(responce)

	case actionAuth:
		var rAuth requestAuth
		err := json.Unmarshal(data, &rAuth)
		if err != nil {
			log.Error("Error parse Auth request  ", err)
			client.senderrorResponse(actionAuth, rType.RequestID, &errorInfo{Number: 400})
			return
		}

		if rAuth.Tokens.Authorization == nil {
			log.Error("Error Tokens.Authorization = nil  ")
			client.senderrorResponse(actionAuth, rAuth.RequestID, &errorInfo{Number: 400})
			return
		}
		responce := client.processAuthRequest(&rAuth)
		client.WriteMessage(responce)

	case actionSubscribe:
		var rSubs requestSubscribe
		err := json.Unmarshal(data, &rSubs)
		if err != nil {
			log.Error("Error parse Subscribe request  ", err)
			client.senderrorResponse(actionSubscribe, rType.RequestID, &errorInfo{Number: 400})
			return
		}
		if rSubs.Filters != nil {
			log.Warn("Filter currently not implemented. Filters will be ignored")
		}
		log.Debug("req subs", rSubs)
		responce := client.processSubscribeRequest(&rSubs)
		client.WriteMessage(responce)

	case actionUnsubscribe:
		var rUnsubs requestUnsubscribe
		err := json.Unmarshal(data, &rUnsubs)
		if err != nil {
			log.Error("Error parse Unsubscribe request  ", err)
			client.senderrorResponse(actionUnsubscribe, rType.RequestID, &errorInfo{Number: 400})
			return
		}
		log.Debug("req Unsubscribe", rUnsubs)
		responce := client.processUnsubscribeRequest(&rUnsubs)
		client.WriteMessage(responce)

	case actionUnsubscribeAll:
		log.Debug("req UnsubscribeAll")
		err := vehicledataprovider.GetInstance().RegestrateUnSubscribAll(client.subscriptionCh)
		if err != nil {
			client.senderrorResponse(actionUnsubscribeAll, rType.RequestID, &errorInfo{Number: 400})
			return
		}
		msg := unsubscribeAllSuccessResponse{Action: actionUnsubscribeAll, RequestID: rType.RequestID, Timestamp: getCurTime()}

		var resp []byte
		resp, err = json.Marshal(msg)
		if err != nil {
			log.Warn("Error marshal json: ", err)
			resp = []byte("Error marshal json")
		}
		client.WriteMessage(resp)

	default:
		log.Error("unsupported action type = ", rType.Action)
		client.senderrorResponse(rType.Action, rType.RequestID, &errorInfo{Number: 400})
	}
}

// process Get request
func (client *WsClientConnection) processGetRequest(request *requestGet) (resp []byte) {
	var err error
	var msg interface{}
	needToAskForData := false
	dataProvider := vehicledataprovider.GetInstance()
	if dataProvider.IsPublicPath(request.Path) == true {
		needToAskForData = true
	} else {
		if client.isAuthorized == false {
			log.Info("Client not Authorized send response 403")
			msg = errorResponse{Action: actionGet, RequestID: request.RequestID, Error: errorInfo{Number: 403}, Timestamp: getCurTime()}
		} else {
			if client.checkPermission(request.Path, getPermission) == true {
				needToAskForData = true
			} else {
				msg = errorResponse{Action: actionGet, RequestID: request.RequestID,
					Error:     errorInfo{Number: 403, Message: "The user is not permitted to access the requested resource"},
					Timestamp: getCurTime()}
			}
		}
	}

	if needToAskForData == true {
		vehData, err := dataProvider.GetDataByPath(request.Path)
		if err != nil {
			log.Warn("No data for path ", request.Path)
			msg = errorResponse{Action: actionGet, RequestID: request.RequestID, Error: errorInfo{Number: 404}, Timestamp: getCurTime()}

		} else {
			log.Debug("Data from dataprovider: %v, request ID: %s", vehData, request.RequestID)
			msg = getSuccessResponse{Action: actionGet, RequestID: request.RequestID, Value: vehData, Timestamp: getCurTime()}
		}
	}

	resp, err = json.Marshal(msg)
	if err != nil {
		log.Warn("Error marshal json: ", err)
		return []byte("Error marshal json")
	}
	return resp
}

// process Set request
func (client *WsClientConnection) processSetRequest(request *requestSet) (resp []byte) {
	var err error
	var msg interface{}
	needToSetData := false
	dataProvider := vehicledataprovider.GetInstance()
	if dataProvider.IsPublicPath(request.Path) == true { //TODO: check for set permission
		needToSetData = true
	} else {
		if client.isAuthorized == false {
			log.Info("Client not Authorized send response 403")
			msg = errorResponse{Action: actionSet, RequestID: request.RequestID, Error: errorInfo{Number: 403}, Timestamp: getCurTime()}
		} else {
			if client.checkPermission(request.Path, setPermission) == true {
				needToSetData = true
			} else {
				msg = errorResponse{Action: actionSet, RequestID: request.RequestID,
					Error:     errorInfo{Number: 403, Message: "The user is not permitted to access the requested resource"},
					Timestamp: getCurTime()}
			}
		}
	}

	if needToSetData == true {
		err := dataProvider.SetDataByPath(request.Path, request.Value)
		if err != nil {
			log.Errorf("Error setting data. Path: %s, err: %s", request.Path, err)
			msg = errorResponse{Action: actionSet, RequestID: request.RequestID, Error: errorInfo{Number: 404}, Timestamp: getCurTime()}

		} else {
			log.Debug("Set Data done path ", request.Path, " value ", request.Value)
			msg = setSuccessResponse{Action: actionSet, RequestID: request.RequestID, Timestamp: getCurTime()}
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

	if client.isAuthorized == false {
		//TODO: add retry count
		//data, errInfo := getPermissionListForClient(*request.Tokens.Authorization)
		data, err := visdbusclient.GetVisPermissionByToken(*request.Tokens.Authorization)
		if err != nil {
			log.Error("Error auth", err)
			msg = errorResponse{Action: actionAuth, RequestID: request.RequestID,
				Error:     errorInfo{Number: 404, Reason: "", Message: err.Error()},
				Timestamp: getCurTime()}
		} else {
			client.permissions = data
			client.isAuthorized = true
			msg = authSuccessResponse{Action: actionAuth, RequestID: request.RequestID, TTL: 10000, Timestamp: getCurTime()}
		}
	} else {
		log.Info("Token ", request.Tokens.Authorization, " already authorized")
		msg = authSuccessResponse{Action: actionAuth, RequestID: request.RequestID, TTL: 10000, Timestamp: getCurTime()}
	}

	resp, err = json.Marshal(msg)
	if err != nil {
		log.Warn("Error marshal json: ", err)
		return []byte("Error marshal json")
	}
	return resp
}

// process Subscribe request
func (client *WsClientConnection) processSubscribeRequest(request *requestSubscribe) (resp []byte) {
	var err error
	var msg interface{}
	isPermissionOK := false
	dataProvider := vehicledataprovider.GetInstance()
	if dataProvider.IsPublicPath(request.Path) == true {
		isPermissionOK = true
	} else {
		if client.isAuthorized == false {
			log.Info("Client not Authorized send response 403 id", request.RequestID)
			msg = errorResponse{Action: actionSubscribe, RequestID: request.RequestID, Error: errorInfo{Number: 403}, Timestamp: getCurTime()}
		} else {
			if client.checkPermission(request.Path, getPermission) == true {
				isPermissionOK = true
			} else {
				msg = errorResponse{Action: actionSubscribe, RequestID: request.RequestID,
					Error:     errorInfo{Number: 403, Message: "The user is not permitted to access the requested resource"},
					Timestamp: getCurTime()}
			}
		}
	}

	if isPermissionOK == true {
		subscrID, err := dataProvider.RegestrateSubscriptionClient(client.subscriptionCh, request.Path)
		if err != nil {
			log.Warn("No data for path ", request.Path)
			msg = errorResponse{Action: actionSubscribe, RequestID: request.RequestID, Error: errorInfo{Number: 404}, Timestamp: getCurTime()}

		} else {
			log.Debug("SubscriptionID from dataprovider ", subscrID)
			msg = subscribeUnsubscribeSuccessResponse{Action: actionSubscribe, RequestID: request.RequestID, SubscriptionID: subscrID, Timestamp: getCurTime()}
		}
	}

	resp, err = json.Marshal(msg)
	if err != nil {
		log.Warn("Error marshal json: ", err)
		return []byte("Error marshal json")
	}
	return resp
}

// process Unsubscribe request
func (client *WsClientConnection) processUnsubscribeRequest(request *requestUnsubscribe) (resp []byte) {
	var err error
	var msg interface{}
	err = vehicledataprovider.GetInstance().RegestrateUnSubscription(client.subscriptionCh, request.SubscriptionID)
	if err != nil {
		log.Warn("Can' unsubscribe from ID", request.SubscriptionID)
		msg = errorResponse{Action: actionUnsubscribe, SubscriptionID: &request.SubscriptionID, RequestID: request.RequestID, Error: errorInfo{Number: 404}, Timestamp: getCurTime()}
	} else {
		log.Debug("UnSubscription from ID ", request.SubscriptionID)
		msg = subscribeUnsubscribeSuccessResponse{Action: actionUnsubscribe, SubscriptionID: request.SubscriptionID, RequestID: request.RequestID, Timestamp: getCurTime()}
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
	msg := errorResponse{Action: action, Timestamp: getCurTime(), Error: *errResp, RequestID: reqID}
	respJSON, err := json.Marshal(msg)
	if err != nil {
		log.Warn("Error marshal json: ", err)
		return
	}
	client.WriteMessage(respJSON)
}

func getCurTime() int64 {
	return time.Now().UnixNano() / 1000000
}
