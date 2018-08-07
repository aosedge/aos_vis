package wsserver

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"gitpct.epam.com/epmd-aepr/aos_vis/dataprovider"
	"gitpct.epam.com/epmd-aepr/aos_vis/dbusclient"
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

type permission uint

/*******************************************************************************
 * Types
 ******************************************************************************/

type wsClient struct {
	wsConnection        *websocket.Conn
	authInfo            *dataprovider.AuthInfo
	dataProvider        *dataprovider.DataProvider
	subscriptionChannel chan dataprovider.SubscriptionOutputData //TODO: change to struct from dataprovider
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

type requestUnsubscribeAll struct {
	Action    string `json:"action"`
	RequestID string `json:"requestId"`
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

type subscribeNotification struct {
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

/*******************************************************************************
 * Private
 ******************************************************************************/

func newClient(wsConnection *websocket.Conn, dataProvider *dataprovider.DataProvider) (client *wsClient, err error) {
	log.WithField("RemoteAddr", wsConnection.RemoteAddr()).Debug("Create new client")

	var localClient wsClient

	localClient.wsConnection = wsConnection
	localClient.subscriptionChannel = make(chan dataprovider.SubscriptionOutputData, 100)
	localClient.dataProvider = dataProvider
	localClient.authInfo = &dataprovider.AuthInfo{}

	client = &localClient

	return client, nil
}

func (client *wsClient) close() (err error) {
	log.WithField("RemoteAddr", client.wsConnection.RemoteAddr()).Debug("Close client")

	client.wsConnection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	return client.wsConnection.Close()
}

func (client *wsClient) processsubscriptionChannel() {
	for {
		data, more := <-client.subscriptionChannel
		if more {
			msg := subscribeNotification{Action: actionSubscription,
				SubscriptionID: data.ID,
				Value:          data.OutData,
				Timestamp:      getCurTime()}
			resp, err := json.Marshal(msg)
			if err != nil {
				log.Warn("Error marshal json: ", err)
				//TODO: create error subscriptionmsg
			}
			err = client.wsConnection.WriteMessage(websocket.TextMessage, resp)
			if err != nil {
				log.Errorf("Error writing message: %s", err)
			}
		} else {
			log.Debug("channelClosed")
			return
		}

	}
}

func (client *wsClient) run() {
	go client.processsubscriptionChannel()

	for {
		mt, message, err := client.wsConnection.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				log.Errorf("Error reading socket: %s", err)
			}
			client.dataProvider.UnsubscribeAll(client.subscriptionChannel)
			close(client.subscriptionChannel)
			break
		}
		if mt == websocket.TextMessage {
			response, err := client.processIncomingMessage(message)
			if err != nil {
				log.Errorf("Error processing message: %s", err)
			}
			err = client.wsConnection.WriteMessage(websocket.TextMessage, response)
			if err != nil {
				log.Errorf("Error writing message: %s", err)
			}
		} else {
			log.WithField("format", mt).Warning("Incoming message in unsupported format")
		}
	}
}

func (client *wsClient) processIncomingMessage(data []byte) (response []byte, err error) {
	log.Debugf("Receive: %s", string(data))

	var rType requestType

	err = json.Unmarshal(data, &rType)
	if err != nil {
		return createErrorResponse("", "", err)
	}

	switch string(rType.Action) {
	case actionGet:
		return client.processGetRequest(data)

	case actionSet:
		return client.processSetRequest(data)

	case actionAuth:
		return client.processAuthRequest(data)

	case actionSubscribe:
		return client.processSubscribeRequest(data)

	case actionUnsubscribe:
		return client.processUnsubscribeRequest(data)

	case actionUnsubscribeAll:
		return client.processUnsubscribeAllRequest(data)

	default:
		return createErrorResponse(rType.Action, rType.RequestID, errors.New("Unsupported action type: "+rType.Action))
	}
}

// process Get request
func (client *wsClient) processGetRequest(requestJSON []byte) (responseJSON []byte, err error) {
	var rGet requestGet

	if err = json.Unmarshal(requestJSON, &rGet); err != nil {
		return createErrorResponse(rGet.Action, rGet.RequestID, err)
	}

	vehData, err := client.dataProvider.GetData(rGet.Path, client.authInfo)
	if err != nil {
		return createErrorResponse(rGet.Action, rGet.RequestID, err)
	}

	log.WithFields(log.Fields{"path": rGet.Path, "value": vehData}).Debug("Get data from dataprovider")

	response := getSuccessResponse{Action: rGet.Action, RequestID: rGet.RequestID, Value: vehData, Timestamp: getCurTime()}
	if responseJSON, err = json.Marshal(response); err != nil {
		return createErrorResponse(rGet.Action, rGet.RequestID, err)
	}

	return responseJSON, nil
}

// process Set request
func (client *wsClient) processSetRequest(requestJSON []byte) (responseJSON []byte, err error) {
	var rSet requestSet

	if err = json.Unmarshal(requestJSON, &rSet); err != nil {
		return createErrorResponse(rSet.Action, rSet.RequestID, err)
	}

	if err = client.dataProvider.SetData(rSet.Path, rSet.Value, client.authInfo); err != nil {
		return createErrorResponse(rSet.Action, rSet.RequestID, err)
	}

	log.WithFields(log.Fields{"path": rSet.Path, "value": rSet.Value}).Debug("Set data to dataprovider")

	response := setSuccessResponse{Action: rSet.Action, RequestID: rSet.RequestID, Timestamp: getCurTime()}
	if responseJSON, err = json.Marshal(response); err != nil {
		return createErrorResponse(rSet.Action, rSet.RequestID, err)
	}

	return responseJSON, nil
}

// process Auth request
func (client *wsClient) processAuthRequest(requestJSON []byte) (responseJSON []byte, err error) {
	var rAuth requestAuth

	if err = json.Unmarshal(requestJSON, &rAuth); err != nil {
		return createErrorResponse(rAuth.Action, rAuth.RequestID, err)
	}

	if rAuth.Tokens.Authorization == nil {
		return createErrorResponse(rAuth.Action, rAuth.RequestID, errors.New("Nil token authorization"))
	}

	if !client.authInfo.IsAuthorized {
		//TODO: add retry count
		//data, errInfo := getPermissionListForClient(*request.Tokens.Authorization)
		data, err := dbusclient.GetVisPermissionByToken(*rAuth.Tokens.Authorization)
		if err != nil {
			return createErrorResponse(rAuth.Action, rAuth.RequestID, err)
		}
		client.authInfo.Permissions = data
		client.authInfo.IsAuthorized = true
	} else {
		log.WithField("token", rAuth.Tokens.Authorization).Debug("Token already authorized")
	}

	response := authSuccessResponse{Action: rAuth.Action, RequestID: rAuth.RequestID, TTL: 10000, Timestamp: getCurTime()}
	if responseJSON, err = json.Marshal(response); err != nil {
		return createErrorResponse(rAuth.Action, rAuth.RequestID, err)
	}

	return responseJSON, nil
}

// process Subscribe request
func (client *wsClient) processSubscribeRequest(requestJSON []byte) (responseJSON []byte, err error) {
	var rSubs requestSubscribe

	if err = json.Unmarshal(requestJSON, &rSubs); err != nil {
		return createErrorResponse(rSubs.Action, rSubs.RequestID, err)
	}

	if rSubs.Filters != nil {
		log.Warn("Filter currently not implemented. Filters will be ignored")
	}

	subscrID, err := client.dataProvider.Subscribe(client.subscriptionChannel, rSubs.Path, client.authInfo)
	if err != nil {
		return createErrorResponse(rSubs.Action, rSubs.RequestID, err)
	}

	log.WithFields(log.Fields{"path": rSubs.Path, "id": subscrID}).Debug("Register subscription")

	response := subscribeUnsubscribeSuccessResponse{Action: actionSubscribe, RequestID: rSubs.RequestID, SubscriptionID: subscrID, Timestamp: getCurTime()}
	if responseJSON, err = json.Marshal(response); err != nil {
		return createErrorResponse(rSubs.Action, rSubs.RequestID, err)
	}

	return responseJSON, nil
}

// process Unsubscribe request
func (client *wsClient) processUnsubscribeRequest(requestJSON []byte) (responseJSON []byte, err error) {
	var rUnsubs requestUnsubscribe

	if err = json.Unmarshal(requestJSON, &rUnsubs); err != nil {
		return createErrorResponse(rUnsubs.Action, rUnsubs.RequestID, errors.New("Client is not authorized"))
	}

	if err = client.dataProvider.Unsubscribe(client.subscriptionChannel, rUnsubs.SubscriptionID); err != nil {
		return createErrorResponse(rUnsubs.Action, rUnsubs.RequestID, err)
	}

	log.WithFields(log.Fields{"id": rUnsubs.SubscriptionID}).Debug("Unregister subscription")

	response := subscribeUnsubscribeSuccessResponse{Action: actionUnsubscribe, SubscriptionID: rUnsubs.SubscriptionID, RequestID: rUnsubs.RequestID, Timestamp: getCurTime()}
	if responseJSON, err = json.Marshal(response); err != nil {
		return createErrorResponse(rUnsubs.Action, rUnsubs.RequestID, err)
	}

	return responseJSON, nil
}

// process UnsubscribeAll request
func (client *wsClient) processUnsubscribeAllRequest(requestJSON []byte) (responseJSON []byte, err error) {
	var rUnsubsAll requestUnsubscribeAll

	err = json.Unmarshal(requestJSON, &rUnsubsAll)
	if err != nil {
		return createErrorResponse(rUnsubsAll.Action, rUnsubsAll.RequestID, errors.New("Client is not authorized"))
	}

	if err = client.dataProvider.UnsubscribeAll(client.subscriptionChannel); err != nil {
		return createErrorResponse(rUnsubsAll.Action, rUnsubsAll.RequestID, err)
	}

	response := unsubscribeAllSuccessResponse{Action: actionUnsubscribeAll, RequestID: rUnsubsAll.RequestID, Timestamp: getCurTime()}
	if responseJSON, err = json.Marshal(response); err != nil {
		return createErrorResponse(rUnsubsAll.Action, rUnsubsAll.RequestID, err)
	}

	return responseJSON, nil
}

func createErrorResponse(action string, reqID string, inErr error) (response []byte, outErr error) {
	code := 400

	switch {
	case strings.Contains(strings.ToLower(inErr.Error()), "not found") ||
		strings.Contains(strings.ToLower(inErr.Error()), "not exist"):
		code = 404
	case strings.Contains(strings.ToLower(inErr.Error()), "not authorized"):
		code = 401
	case strings.Contains(strings.ToLower(inErr.Error()), "not have permissions"):
		code = 403
	}

	info := errorInfo{Number: code, Message: inErr.Error()}
	response, outErr = json.Marshal(errorResponse{Action: action, Timestamp: getCurTime(), Error: info, RequestID: reqID})
	if outErr != nil {
		log.Errorf("Error creating error response: %s", outErr)
	}

	outErr = inErr

	return response, outErr
}

func getCurTime() int64 {
	return time.Now().UnixNano() / 1000000
}
