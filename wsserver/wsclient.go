package wsserver

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
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
	wsConnection      *websocket.Conn
	authInfo          *dataprovider.AuthInfo
	dataProvider      *dataprovider.DataProvider
	subscribeChannels map[uint64]<-chan interface{}
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

type subscribeError struct {
	Action         string    `json:"action"`
	SubscriptionID string    `json:"subscriptionId"`
	Error          errorInfo `json:"error"`
	Timestamp      int64     `json:"timestamp"`
}

type errorResponse struct {
	Action    string    `json:"action"`
	RequestID string    `json:"requestId"`
	Error     errorInfo `json:"error"`
	Timestamp int64     `json:"timestamp"`
}

//TODO: add map error number message
type errorInfo struct {
	Number  int
	Reason  string
	Message string
}

/*******************************************************************************
 * Variables
 ******************************************************************************/

var mutex sync.Mutex

/*******************************************************************************
 * Private
 ******************************************************************************/

func newClient(wsConnection *websocket.Conn, dataProvider *dataprovider.DataProvider) (client *wsClient, err error) {
	log.WithField("RemoteAddr", wsConnection.RemoteAddr()).Info("Create new client")

	var localClient wsClient

	localClient.wsConnection = wsConnection
	localClient.subscribeChannels = make(map[uint64]<-chan interface{})
	localClient.dataProvider = dataProvider
	localClient.authInfo = &dataprovider.AuthInfo{}

	client = &localClient

	return client, nil
}

func (client *wsClient) close() (err error) {
	log.WithField("RemoteAddr", client.wsConnection.RemoteAddr()).Info("Close client")

	mutex.Lock()
	client.wsConnection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	mutex.Unlock()

	client.unsubscribeAll()

	return client.wsConnection.Close()
}

func (client *wsClient) run() {
	for {
		mt, message, err := client.wsConnection.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure) &&
				!strings.Contains(err.Error(), "use of closed network connection") {
				log.Errorf("Error reading socket: %s", err)
			}

			break
		}
		if mt == websocket.TextMessage {
			log.Debugf("Receive: %s", string(message))

			response, err := client.processIncomingMessage(message)
			if err != nil {
				log.Errorf("Error processing message: %s", err)
			}

			log.Debugf("Send: %s", string(response))

			mutex.Lock()
			err = client.wsConnection.WriteMessage(websocket.TextMessage, response)
			mutex.Unlock()
			if err != nil {
				log.Errorf("Error writing message: %s", err)
			}
		} else {
			log.WithField("format", mt).Warning("Incoming message in unsupported format")
		}
	}
}

func (client *wsClient) processIncomingMessage(data []byte) (response []byte, err error) {
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

	response := getSuccessResponse{
		Action:    rGet.Action,
		RequestID: rGet.RequestID,
		Value:     vehData,
		Timestamp: getCurTime()}

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

	response := setSuccessResponse{
		Action:    rSet.Action,
		RequestID: rSet.RequestID,
		Timestamp: getCurTime()}

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

	response := authSuccessResponse{
		Action:    rAuth.Action,
		RequestID: rAuth.RequestID,
		TTL:       10000,
		Timestamp: getCurTime()}

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

	subscribeID, channel, err := client.dataProvider.Subscribe(rSubs.Path, client.authInfo)
	if err != nil {
		return createErrorResponse(rSubs.Action, rSubs.RequestID, err)
	}

	log.WithFields(log.Fields{"path": rSubs.Path, "id": subscribeID}).Debug("Register subscription")

	response := subscribeUnsubscribeSuccessResponse{
		Action:         actionSubscribe,
		RequestID:      rSubs.RequestID,
		SubscriptionID: strconv.FormatUint(subscribeID, 10),
		Timestamp:      getCurTime()}

	if responseJSON, err = json.Marshal(response); err != nil {
		return createErrorResponse(rSubs.Action, rSubs.RequestID, err)
	}

	client.subscribeChannels[subscribeID] = channel
	go client.processSubscribeChannel(subscribeID, channel)

	return responseJSON, nil
}

// process Unsubscribe request
func (client *wsClient) processUnsubscribeRequest(requestJSON []byte) (responseJSON []byte, err error) {
	var rUnsubs requestUnsubscribe

	if err = json.Unmarshal(requestJSON, &rUnsubs); err != nil {
		return createErrorResponse(rUnsubs.Action, rUnsubs.RequestID, err)
	}

	subscribeID, err := strconv.ParseUint(rUnsubs.SubscriptionID, 10, 64)
	if err != nil {
		return createErrorResponse(rUnsubs.Action, rUnsubs.RequestID, err)
	}

	if err = client.dataProvider.Unsubscribe(subscribeID, client.authInfo); err != nil {
		return createErrorResponse(rUnsubs.Action, rUnsubs.RequestID, err)
	}

	delete(client.subscribeChannels, subscribeID)

	log.WithFields(log.Fields{"id": rUnsubs.SubscriptionID}).Debug("Unregister subscription")

	response := subscribeUnsubscribeSuccessResponse{
		Action:         actionUnsubscribe,
		SubscriptionID: rUnsubs.SubscriptionID,
		RequestID:      rUnsubs.RequestID,
		Timestamp:      getCurTime()}

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
		return createErrorResponse(rUnsubsAll.Action, rUnsubsAll.RequestID, err)
	}

	if err = client.unsubscribeAll(); err != nil {
		return createErrorResponse(rUnsubsAll.Action, rUnsubsAll.RequestID, err)
	}

	response := unsubscribeAllSuccessResponse{
		Action:    actionUnsubscribeAll,
		RequestID: rUnsubsAll.RequestID,
		Timestamp: getCurTime()}

	if responseJSON, err = json.Marshal(response); err != nil {
		return createErrorResponse(rUnsubsAll.Action, rUnsubsAll.RequestID, err)
	}

	return responseJSON, nil
}

func (client *wsClient) processSubscribeChannel(id uint64, channel <-chan interface{}) {
	for {
		data, more := <-channel
		if more {
			notification := subscribeNotification{
				Action:         actionSubscription,
				SubscriptionID: strconv.FormatUint(id, 10),
				Value:          data,
				Timestamp:      getCurTime()}

			notificationJSON, err := json.Marshal(notification)
			if err != nil {
				notificationJSON, err = createSubscribeError(id, err)
				if err != nil {
					log.Errorf("Can't create subscribe error response: %s", err)
					break
				}
			}

			log.Debugf("Send: %s", string(notificationJSON))

			mutex.Lock()
			err = client.wsConnection.WriteMessage(websocket.TextMessage, notificationJSON)
			mutex.Unlock()
			if err != nil {
				log.Errorf("Error writing message: %s", err)
			}
		} else {
			log.WithField("subscribeID", id).Debug("Subscription closed")
			return
		}

	}
}

func (client *wsClient) unsubscribeAll() (err error) {
	for subscribeID := range client.subscribeChannels {
		if localErr := client.dataProvider.Unsubscribe(subscribeID, client.authInfo); localErr != nil {
			err = localErr
		}
	}

	client.subscribeChannels = make(map[uint64]<-chan interface{})

	return err
}

func codeFromError(err error) (code int) {
	switch {
	case strings.Contains(strings.ToLower(err.Error()), "not found") ||
		strings.Contains(strings.ToLower(err.Error()), "not exist"):
		code = 404
	case strings.Contains(strings.ToLower(err.Error()), "not authorized"):
		code = 401
	case strings.Contains(strings.ToLower(err.Error()), "not have permissions"):
		code = 403
	default:
		code = 400
	}

	return code
}

func createErrorResponse(action string, reqID string, inErr error) (responseJSON []byte, outErr error) {
	response := errorResponse{
		Action:    action,
		Timestamp: getCurTime(),
		Error:     errorInfo{Number: codeFromError(inErr), Message: inErr.Error()},
		RequestID: reqID}

	responseJSON, outErr = json.Marshal(response)
	if outErr != nil {
		log.Errorf("Error creating error response: %s", outErr)
	}

	outErr = inErr

	return responseJSON, outErr
}

func createSubscribeError(id uint64, inErr error) (responseJSON []byte, err error) {
	response := subscribeError{
		Action:         actionSubscription,
		SubscriptionID: strconv.FormatUint(id, 10),
		Error:          errorInfo{Number: codeFromError(inErr), Message: inErr.Error()},
		Timestamp:      getCurTime(),
	}

	responseJSON, err = json.Marshal(response)
	if err != nil {
		return responseJSON, err
	}

	return responseJSON, nil
}

func getCurTime() int64 {
	return time.Now().UnixNano() / 1000000
}
