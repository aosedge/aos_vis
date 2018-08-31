package wsserver_test

import (
	"encoding/json"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/godbus/dbus"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"gitpct.epam.com/epmd-aepr/aos_vis/config"
	"gitpct.epam.com/epmd-aepr/aos_vis/wsserver"
)

const serverURL = "localhost:8088"

type visResponse struct {
	Action         string      `json:"action"`
	RequestID      string      `json:"requestId"`
	Value          interface{} `json:"value"`
	Error          *errorInfo  `json:"error"`
	Ttl            int64       `json:"TTL"`
	SubscriptionID *string     `json:"subscriptionId"`
	Timestamp      int64       `json:"timestamp"`
}

type errorInfo struct {
	Number  int
	Reason  string
	Message string
}

type dbusInterface struct {
}

type configuration struct {
	ServerUrl string
	VISCert   string
	VISKey    string
}

/*******************************************************************************
 * Init
 ******************************************************************************/

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		TimestampFormat:  "2006-01-02 15:04:05.000",
		FullTimestamp:    true})
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
}

func (GetPermission dbusInterface) GetPermission(token string) (string, string, *dbus.Error) {
	return `{"Signal.*": "rw"}`, "OK", nil
}

/*******************************************************************************
 * Main
 ******************************************************************************/

func TestMain(m *testing.M) {
	conn, err := dbus.SessionBus()
	if err != nil {
		log.Fatalf("Can't create session connection: %s", err)
	}

	reply, err := conn.RequestName("com.aosservicemanager.vistoken", dbus.NameFlagDoNotQueue)
	if err != nil {
		log.Fatal("Can't request name")
	}

	if reply != dbus.RequestNameReplyPrimaryOwner {
		log.Fatal("Name already taken")
	}

	dbusserver := dbusInterface{}
	conn.Export(dbusserver, "/com/aosservicemanager/vistoken", "com.aosservicemanager.vistoken")

	configJSON := `{
		"VISCert": "../data/wwwivi.crt.pem",
		"VISKey":  "../data/wwwivi.key.pem",
		"Adapters":[
			{
				"Plugin":"../storageadapter.so",
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
	if err = decoder.Decode(&cfg); err != nil {
		log.Fatalf("Can't parse config: %s", err)
	}

	cfg.ServerURL = serverURL

	server, err := wsserver.New(&cfg)
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

func closeConnection(c *websocket.Conn) {
	c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	c.Close()
}

func TestGetNoAuth(t *testing.T) {
	u := url.URL{Scheme: "wss", Host: serverURL, Path: "/"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Can't connect to ws server %s", err)
	}
	defer closeConnection(c)

	sendRequest(t, c, map[string]interface{}{
		"action":    "get",
		"path":      "Attribute.Vehicle.VehicleIdentification.VIN",
		"requestId": "8765"},
		false)
}

func TestGet(t *testing.T) {
	u := url.URL{Scheme: "wss", Host: serverURL, Path: "/"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Can't connect to ws server %s", err)
	}
	defer closeConnection(c)

	response := sendRequest(t, c, map[string]interface{}{
		"action":    "get",
		"path":      "Signal.Drivetrain.InternalCombustionEngine.RPM",
		"requestId": "8755"},
		true)

	if response.Error == nil || response.Error.Number != 401 {
		t.Fatalf("Should be error 401")
	}

	sendRequest(t, c, map[string]interface{}{
		"action":    "authorize",
		"tokens":    map[string]interface{}{"authorization": "appUID"},
		"requestId": "12345"},
		false)

	response = sendRequest(t, c, map[string]interface{}{
		"action":    "get",
		"path":      "Signal.Drivetrain.InternalCombustionEngine.RPM",
		"requestId": "12347"},
		false)
}

func TestSet(t *testing.T) {
	u := url.URL{Scheme: "wss", Host: serverURL, Path: "/"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Can't connect to ws server %s", err)
	}
	defer closeConnection(c)

	sendRequest(t, c, map[string]interface{}{
		"action":    "authorize",
		"tokens":    map[string]interface{}{"authorization": "appUID"},
		"requestId": "12345"},
		false)

	response := sendRequest(t, c, map[string]interface{}{
		"action": "set",
		"path":   "Signal.Cabin.Door.Row1.*",
		"value": []interface{}{
			map[string]interface{}{"Right.IsLocked": true},
			map[string]interface{}{"Right.Window.Position": 100},
			map[string]interface{}{"Left.IsLocked": true},
			map[string]interface{}{"Left.Window.Position": 250}},
		"requestId": "8888"},
		false)

	if response.Error != nil {
		t.Fatalf("Error parsing get request: %s", response.Error.Message)
	}
}

func TestSubscribeUnsubscribe(t *testing.T) {

	u := url.URL{Scheme: "wss", Host: serverURL, Path: "/"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Can't connect to ws server: %s", err)
	}
	defer closeConnection(c)

	sendRequest(t, c, map[string]interface{}{
		"action":    "authorize",
		"tokens":    map[string]interface{}{"authorization": "appUID"},
		"requestId": "12345"},
		false)

	// Subscribe

	response := sendRequest(t, c, map[string]interface{}{
		"action":    "subscribe",
		"path":      "Signal.Cabin.Door.Row1.Right.Window.Position",
		"requestId": "1004"},
		false)

	if response.SubscriptionID == nil {
		t.Fatalf("No subscriptionId")
	}

	subscribeID := *response.SubscriptionID

	// Change data

	response = sendRequest(t, c, map[string]interface{}{
		"action":    "set",
		"path":      "Signal.Cabin.Door.Row1.Right.Window.Position",
		"value":     123,
		"requestId": "1004"},
		false)

	// Wait for notification

	c.SetReadDeadline(time.Now().Add(1 * time.Second))

	var notification visResponse

	_, message, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message from server: %s", err)
	}
	if err = json.Unmarshal(message, &notification); err != nil {
		t.Fatalf("Error parsing notification: %s", err)
	}

	if notification.Action != "subscription" || *notification.SubscriptionID != subscribeID || notification.Value.(float64) != 123.0 {
		t.Fatalf("Unexpected value")
	}
	if notification.Error != nil {
		t.Fatalf("Unexpected error for subscription: %s", notification.Error.Message)
	}

	// Unsubscribe wrong id

	response = sendRequest(t, c, map[string]interface{}{
		"action":         "unsubscribe",
		"subscriptionId": "1",
		"requestId":      "1004"},
		true)

	if response.Error == nil {
		t.Fatal("Unexpected positive response")
	}

	// Unsubscribe

	response = sendRequest(t, c, map[string]interface{}{
		"action":         "unsubscribe",
		"subscriptionId": "0",
		"requestId":      "1004"},
		false)

	// UnsubscribeAll

	response = sendRequest(t, c, map[string]interface{}{
		"action":    "unsubscribeAll",
		"requestId": "1004"},
		false)
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func sendRequest(t *testing.T, c *websocket.Conn, request map[string]interface{}, ignoreReceiveError bool) (response visResponse) {
	message, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Error parsing request: %s", err)
	}

	if err := c.WriteMessage(websocket.TextMessage, message); err != nil {
		t.Fatalf("Can't send message to server %s", err)
	}

	_, message, err = c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message from server: %s", err)
	}

	if err = json.Unmarshal(message, &response); err != nil {
		t.Fatalf("Error parsing response: %s", err)
	}

	if response.Action != request["action"] || response.RequestID != request["requestId"] {
		t.Fatalf("Unexpected value")
	}

	if !ignoreReceiveError && response.Error != nil {
		t.Fatalf("Unexpected error for %s: %s", request["action"], response.Error.Message)
	}

	return response
}
