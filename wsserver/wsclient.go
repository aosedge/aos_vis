package wsserver

import (
	"encoding/json"
	"errors"
	"fmt"
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

// VIS actions
const (
	ActionGet            = "get"
	ActionSet            = "set"
	ActionAuth           = "authorize"
	ActionSubscribe      = "subscribe"
	ActionUnsubscribe    = "unsubscribe"
	ActionUnsubscribeAll = "unsubscribeAll"
	ActionSubscription   = "subscription"
)

const (
	writeSocketTimeout = 10 * time.Second
)

/*******************************************************************************
 * Types
 ******************************************************************************/

type wsClient struct {
	wsConnection      *websocket.Conn
	authInfo          *dataprovider.AuthInfo
	dataProvider      *dataprovider.DataProvider
	subscribeChannels map[uint64]<-chan interface{}
	sync.Mutex
}

// MessageHeader VIS message header
type MessageHeader struct {
	Action    string `json:"action"`
	RequestID string `json:"requestId"`
}

// ErrorInfo VIS error info
type ErrorInfo struct {
	Number  int    `json:"number"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// Tokens VIS authorize tokens
type Tokens struct {
	Authorization    string `json:"authorization,omitempty"`
	WwwVehicleDevice string `json:"www-vehicle-device,omitempty"`
}

// AuthRequest VIS authorize request
type AuthRequest struct {
	MessageHeader
	Tokens Tokens `json:"tokens"`
}

// AuthResponse VIS authorize success response
type AuthResponse struct {
	MessageHeader
	Error *ErrorInfo `json:"error,omitempty"`
	TTL   int64      `json:"TTL,omitempty"`
}

// GetRequest VIS get request
type GetRequest struct {
	MessageHeader
	Path string `json:"path"`
}

// GetResponse VIS get success response
type GetResponse struct {
	MessageHeader
	Error     *ErrorInfo  `json:"error,omitempty"`
	Value     interface{} `json:"value,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// SetRequest VIS set request
type SetRequest struct {
	MessageHeader
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

// SetResponse VIS set success response
type SetResponse struct {
	MessageHeader
	Error     *ErrorInfo `json:"error,omitempty"`
	Timestamp int64      `json:"timestamp,omitempty"`
}

// SubscribeRequest VIS subscribe request
type SubscribeRequest struct {
	MessageHeader
	Path    string `json:"path"`
	Filters string `json:"filters,omitempty"` //TODO: will be implemented later
}

// SubscribeResponse VIS subscribe success response
type SubscribeResponse struct {
	MessageHeader
	Error          *ErrorInfo `json:"error,omitempty"`
	SubscriptionID string     `json:"subscriptionId,omitempty"`
	Timestamp      int64      `json:"timestamp"`
}

// SubscriptionNotification VIS subscription notification
type SubscriptionNotification struct {
	Error          *ErrorInfo  `json:"error,omitempty"`
	Action         string      `json:"action"`
	SubscriptionID string      `json:"subscriptionId"`
	Value          interface{} `json:"value,omitempty"`
	Timestamp      int64       `json:"timestamp"`
}

// UnsubscribeRequest VIS unsubscribe request
type UnsubscribeRequest struct {
	MessageHeader
	SubscriptionID string `json:"subscriptionId"`
}

// UnsubscribeResponse VIS unsubscribe success response
type UnsubscribeResponse struct {
	MessageHeader
	Error          *ErrorInfo `json:"error,omitempty"`
	SubscriptionID string     `json:"subscriptionId"`
	Timestamp      int64      `json:"timestamp"`
}

// UnsubscribeAllRequest VIS unsubscribe all request
type UnsubscribeAllRequest struct {
	MessageHeader
}

// UnsubscribeAllResponse VIS unsubscribe all success response
type UnsubscribeAllResponse struct {
	MessageHeader
	Error     *ErrorInfo `json:"error,omitempty"`
	Timestamp int64      `json:"timestamp"`
}

/*******************************************************************************
 * Variables
 ******************************************************************************/

/*******************************************************************************
 * Private
 ******************************************************************************/

func newClient(wsConnection *websocket.Conn, dataProvider *dataprovider.DataProvider) (client *wsClient, err error) {
	log.WithField("RemoteAddr", wsConnection.RemoteAddr()).Info("Create new client")

	client = &wsClient{
		wsConnection:      wsConnection,
		subscribeChannels: make(map[uint64]<-chan interface{}),
		dataProvider:      dataProvider,
		authInfo:          &dataprovider.AuthInfo{}}

	return client, nil
}

func (client *wsClient) close(sendCloseMessage bool) (err error) {
	log.WithField("RemoteAddr", client.wsConnection.RemoteAddr()).Info("Close client")

	client.unsubscribeAll()

	if sendCloseMessage {
		client.sendMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	}

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

			if response != nil {
				client.sendMessage(websocket.TextMessage, response)
			}
		} else {
			log.WithField("format", mt).Warning("Incoming message in unsupported format")
		}
	}
}

func (client *wsClient) processIncomingMessage(data []byte) (responseJSON []byte, err error) {
	var header MessageHeader

	if err = json.Unmarshal(data, &header); err != nil {
		return nil, err
	}

	var response interface{}

	switch string(header.Action) {
	case ActionGet:
		response, err = client.processGetRequest(data)

	case ActionSet:
		response, err = client.processSetRequest(data)

	case ActionAuth:
		response, err = client.processAuthRequest(data)

	case ActionSubscribe:
		response, err = client.processSubscribeRequest(data)

	case ActionUnsubscribe:
		response, err = client.processUnsubscribeRequest(data)

	case ActionUnsubscribeAll:
		response, err = client.processUnsubscribeAllRequest(data)

	default:
		err = fmt.Errorf("Unsupported action type: %s", header.Action)
	}

	if err != nil {
		return nil, err
	}

	if responseJSON, err = json.Marshal(response); err != nil {
		return nil, err
	}

	return responseJSON, nil
}

// process Get request
func (client *wsClient) processGetRequest(requestJSON []byte) (responseItf interface{}, err error) {
	var request GetRequest

	if err = json.Unmarshal(requestJSON, &request); err != nil {
		return nil, err
	}

	response := GetResponse{
		MessageHeader: request.MessageHeader,
		Timestamp:     getCurTime()}

	vehicleData, err := client.dataProvider.GetData(request.Path, client.authInfo)
	if err != nil {
		response.Error = createErrorInfo(err)
		return &response, nil
	}

	response.Value = vehicleData

	return &response, nil
}

// process Set request
func (client *wsClient) processSetRequest(requestJSON []byte) (responseItf interface{}, err error) {
	var request SetRequest

	if err = json.Unmarshal(requestJSON, &request); err != nil {
		return nil, err
	}

	response := SetResponse{
		MessageHeader: request.MessageHeader,
		Timestamp:     getCurTime()}

	if err = client.dataProvider.SetData(request.Path, request.Value, client.authInfo); err != nil {
		response.Error = createErrorInfo(err)
		return &response, nil
	}

	return &response, nil
}

// process Auth request
func (client *wsClient) processAuthRequest(requestJSON []byte) (responseItf interface{}, err error) {
	var request AuthRequest

	if err = json.Unmarshal(requestJSON, &request); err != nil {
		return nil, err
	}

	response := AuthResponse{
		MessageHeader: request.MessageHeader}

	if request.Tokens.Authorization == "" {
		response.Error = createErrorInfo(errors.New("empty token authorization"))
		return &response, nil
	}

	if client.authInfo.Permissions, err = dbusclient.GetVisPermissionByToken(request.Tokens.Authorization); err != nil {
		response.Error = createErrorInfo(errors.New("empty token authorization"))
		return &response, nil
	}

	client.authInfo.IsAuthorized = true
	response.TTL = 10000

	return &response, nil
}

// process Subscribe request
func (client *wsClient) processSubscribeRequest(requestJSON []byte) (responseItf interface{}, err error) {
	var request SubscribeRequest

	if err = json.Unmarshal(requestJSON, &request); err != nil {
		return nil, err
	}

	if request.Filters != "" {
		log.Warn("Filter currently not implemented. Filters will be ignored")
	}

	response := SubscribeResponse{
		MessageHeader: request.MessageHeader,
		Timestamp:     getCurTime()}

	id, channel, err := client.dataProvider.Subscribe(request.Path, client.authInfo)
	if err != nil {
		response.Error = createErrorInfo(err)
		return &response, nil
	}

	log.WithFields(log.Fields{"path": request.Path, "id": id}).Debug("Register subscription")

	response.SubscriptionID = strconv.FormatUint(id, 10)

	client.subscribeChannels[id] = channel
	go client.processSubscribeChannel(id, channel)

	return &response, nil
}

// process Unsubscribe request
func (client *wsClient) processUnsubscribeRequest(requestJSON []byte) (responseItf interface{}, err error) {
	var request UnsubscribeRequest

	if err = json.Unmarshal(requestJSON, &request); err != nil {
		return nil, err
	}

	response := UnsubscribeResponse{
		MessageHeader:  request.MessageHeader,
		SubscriptionID: request.SubscriptionID,
		Timestamp:      getCurTime()}

	subscribeID, err := strconv.ParseUint(request.SubscriptionID, 10, 64)
	if err != nil {
		response.Error = createErrorInfo(err)
		return &response, nil
	}

	if err = client.dataProvider.Unsubscribe(subscribeID, client.authInfo); err != nil {
		response.Error = createErrorInfo(err)
		return &response, nil
	}

	delete(client.subscribeChannels, subscribeID)

	log.WithFields(log.Fields{"id": request.SubscriptionID}).Debug("Unregister subscription")

	return &response, nil
}

// process UnsubscribeAll request
func (client *wsClient) processUnsubscribeAllRequest(requestJSON []byte) (responseItf interface{}, err error) {
	var request UnsubscribeAllRequest

	if err = json.Unmarshal(requestJSON, &request); err != nil {
		return nil, err
	}

	response := UnsubscribeAllResponse{
		MessageHeader: request.MessageHeader,
		Timestamp:     getCurTime()}

	if err = client.unsubscribeAll(); err != nil {
		response.Error = createErrorInfo(err)
		return &response, nil
	}

	return &response, nil
}

func (client *wsClient) processSubscribeChannel(id uint64, channel <-chan interface{}) {
	for {
		data, more := <-channel
		if more {
			subscriptionID := strconv.FormatUint(id, 10)

			notification := SubscriptionNotification{
				Action:         ActionSubscription,
				SubscriptionID: subscriptionID,
				Value:          data,
				Timestamp:      getCurTime()}

			notificationJSON, err := json.Marshal(notification)
			if err != nil {
				log.Errorf("Can't marshal subscription notification: %s", err)
			}

			if notificationJSON != nil {
				client.sendMessage(websocket.TextMessage, notificationJSON)
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

func createErrorInfo(err error) (errorInfo *ErrorInfo) {
	if err == nil {
		return nil
	}

	errorInfo = &ErrorInfo{Message: err.Error()}

	switch {
	case strings.Contains(strings.ToLower(err.Error()), "not found") ||
		strings.Contains(strings.ToLower(err.Error()), "not exist"):
		errorInfo.Number = 404
	case strings.Contains(strings.ToLower(err.Error()), "not authorized"):
		errorInfo.Number = 401
	case strings.Contains(strings.ToLower(err.Error()), "not have permissions"):
		errorInfo.Number = 403
	default:
		errorInfo.Number = 400
	}

	return errorInfo
}

func getCurTime() int64 {
	return time.Now().UnixNano() / 1000000
}

func (client *wsClient) sendMessage(messageType int, data []byte) (err error) {
	client.Lock()
	defer client.Unlock()

	log.Debugf("Send: %s", string(data))

	if writeSocketTimeout != 0 {
		client.wsConnection.SetWriteDeadline(time.Now().Add(writeSocketTimeout))
	}

	if err = client.wsConnection.WriteMessage(messageType, data); err != nil {
		log.Errorf("Can't write message: %s", err)

		client.wsConnection.Close()

		return err
	}

	return nil
}
