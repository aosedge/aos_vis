package visserver_test

import (
	"encoding/json"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"gitpct.epam.com/epmd-aepr/aos_servicemanager/wsclient"

	"github.com/godbus/dbus"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"gitpct.epam.com/epmd-aepr/aos_vis/config"
	"gitpct.epam.com/epmd-aepr/aos_vis/visserver"
)

const serverURL = "wss://localhost:8088"

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

	url, err := url.Parse(serverURL)
	if err != nil {
		log.Fatalf("Can't parse url: %s", err)
	}

	cfg.ServerURL = url.Host

	server, err := visserver.New(&cfg)
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
	client, err := wsclient.New("TestClient", nil)
	if err != nil {
		t.Fatalf("Can't create client: %s", err)
	}
	defer client.Close()

	if err = client.Connect(serverURL); err != nil {
		t.Fatalf("Can't connect to server: %s", err)
	}

	getRequest := visserver.GetRequest{
		MessageHeader: visserver.MessageHeader{
			Action:    visserver.ActionGet,
			RequestID: "8765"},
		Path: "Attribute.Vehicle.VehicleIdentification.VIN"}
	getResponse := visserver.GetResponse{}

	if err = client.SendRequest("RequestID", &getRequest, &getResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if getResponse.Error != nil {
		t.Fatalf("Get request error: %s", getResponse.Error.Message)
	}
}

func TestGet(t *testing.T) {
	client, err := wsclient.New("TestClient", nil)
	if err != nil {
		t.Fatalf("Can't create client: %s", err)
	}
	defer client.Close()

	if err = client.Connect(serverURL); err != nil {
		t.Fatalf("Can't connect to server: %s", err)
	}

	getRequest := visserver.GetRequest{
		MessageHeader: visserver.MessageHeader{
			Action:    visserver.ActionGet,
			RequestID: "8755"},
		Path: "Signal.Drivetrain.InternalCombustionEngine.RPM"}
	getResponse := visserver.GetResponse{}

	if err = client.SendRequest("RequestID", &getRequest, &getResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if getResponse.Error == nil || getResponse.Error.Number != 401 {
		t.Fatalf("Should be error 401")
	}

	authRequest := visserver.AuthRequest{
		MessageHeader: visserver.MessageHeader{
			Action:    visserver.ActionAuth,
			RequestID: "12345"},
		Tokens: visserver.Tokens{
			Authorization: "appUID"}}
	authResponse := visserver.AuthResponse{}

	if err = client.SendRequest("RequestID", &authRequest, &authResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if authResponse.Error != nil {
		t.Fatalf("Auth request error: %s", authResponse.Error.Message)
	}

	if err = client.SendRequest("RequestID", &getRequest, &getResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if authResponse.Error != nil {
		t.Fatalf("Get request error: %s", authResponse.Error.Message)
	}
}

func TestSet(t *testing.T) {
	client, err := wsclient.New("TestClient", nil)
	if err != nil {
		t.Fatalf("Can't create client: %s", err)
	}
	defer client.Close()

	if err = client.Connect(serverURL); err != nil {
		t.Fatalf("Can't connect to server: %s", err)
	}

	authRequest := visserver.AuthRequest{
		MessageHeader: visserver.MessageHeader{
			Action:    visserver.ActionAuth,
			RequestID: "12345"},
		Tokens: visserver.Tokens{
			Authorization: "appUID"}}
	authResponse := visserver.AuthResponse{}

	if err = client.SendRequest("RequestID", &authRequest, &authResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if authResponse.Error != nil {
		t.Fatalf("Auth request error: %s", authResponse.Error.Message)
	}

	setRequest := visserver.SetRequest{
		MessageHeader: visserver.MessageHeader{
			Action:    visserver.ActionSet,
			RequestID: "8888"},
		Path: "Signal.Cabin.Door.Row1.*",
		Value: []interface{}{
			map[string]interface{}{"Right.IsLocked": true},
			map[string]interface{}{"Right.Window.Position": 100},
			map[string]interface{}{"Left.IsLocked": true},
			map[string]interface{}{"Left.Window.Position": 250}}}
	setResponse := visserver.GetResponse{}

	if err = client.SendRequest("RequestID", &setRequest, &setResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if setResponse.Error != nil {
		t.Fatalf("Set request error: %s", setResponse.Error.Message)
	}
}

func TestSubscribeUnsubscribe(t *testing.T) {
	notificationChannel := make(chan visserver.SubscriptionNotification, 1)

	client, err := wsclient.New("TestClient", func(data []byte) {
		var notification visserver.SubscriptionNotification

		if err := json.Unmarshal(data, &notification); err != nil {
			t.Fatalf("Error parsing notification: %s", err)
		}

		notificationChannel <- notification
	})
	if err != nil {
		t.Fatalf("Can't create client: %s", err)
	}
	defer client.Close()

	if err = client.Connect(serverURL); err != nil {
		t.Fatalf("Can't connect to server: %s", err)
	}

	authRequest := visserver.AuthRequest{
		MessageHeader: visserver.MessageHeader{
			Action:    visserver.ActionAuth,
			RequestID: "12345"},
		Tokens: visserver.Tokens{
			Authorization: "appUID"}}
	authResponse := visserver.AuthResponse{}

	if err = client.SendRequest("RequestID", &authRequest, &authResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if authResponse.Error != nil {
		t.Fatalf("Auth request error: %s", authResponse.Error.Message)
	}

	// Subscribe

	subscribeRequest := visserver.SubscribeRequest{
		MessageHeader: visserver.MessageHeader{
			Action:    visserver.ActionSubscribe,
			RequestID: "1004"},
		Path: "Signal.Cabin.Door.Row1.Right.Window.Position"}
	subscribeResponse := visserver.SubscribeResponse{}

	if err = client.SendRequest("RequestID", &subscribeRequest, &subscribeResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if subscribeResponse.Error != nil {
		t.Fatalf("Subscribe request error: %s", authResponse.Error.Message)
	}

	if subscribeResponse.SubscriptionID == "" {
		t.Fatalf("No subscriptionId")
	}

	subscriptionID := subscribeResponse.SubscriptionID

	// Change data

	setRequest := visserver.SetRequest{
		MessageHeader: visserver.MessageHeader{
			Action:    visserver.ActionSet,
			RequestID: "1004"},
		Path:  "Signal.Cabin.Door.Row1.Right.Window.Position",
		Value: 123}
	setResponse := visserver.GetResponse{}

	if err = client.SendRequest("RequestID", &setRequest, &setResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if setResponse.Error != nil {
		t.Fatalf("Set request error: %s", setResponse.Error.Message)
	}

	// Wait for notification

	select {
	case notification := <-notificationChannel:
		if notification.Action != "subscription" || notification.SubscriptionID != subscriptionID || notification.Value.(float64) != 123.0 {
			t.Fatalf("Unexpected value")
		}
		if notification.Error != nil {
			t.Fatalf("Unexpected error for subscription: %s", notification.Error.Message)
		}

	case <-time.After(1 * time.Second):
		t.Fatal("Waiting for subscription notification timeout")
	}

	// Unsubscribe wrong id

	unsubscribeRequest := visserver.UnsubscribeRequest{
		MessageHeader: visserver.MessageHeader{
			Action:    visserver.ActionUnsubscribe,
			RequestID: "1004"},
		SubscriptionID: "1"}
	unsubscribeResponse := visserver.UnsubscribeResponse{}

	if err = client.SendRequest("RequestID", &unsubscribeRequest, &unsubscribeResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if unsubscribeResponse.Error == nil {
		t.Fatal("Unexpected positive response")
	}

	// Unsubscribe

	unsubscribeRequest.SubscriptionID = subscriptionID
	unsubscribeResponse = visserver.UnsubscribeResponse{}

	if err = client.SendRequest("RequestID", &unsubscribeRequest, &unsubscribeResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if unsubscribeResponse.Error != nil {
		t.Fatalf("Unsubscribe request error: %s", unsubscribeResponse.Error.Message)
	}

	// UnsubscribeAll

	unsubscribeAllRequest := visserver.UnsubscribeAllRequest{
		MessageHeader: visserver.MessageHeader{
			Action: visserver.ActionUnsubscribeAll}}
	unsubscribeAllResponse := visserver.UnsubscribeAllResponse{}

	if err = client.SendRequest("RequestID", &unsubscribeAllRequest, &unsubscribeAllResponse); err != nil {
		t.Errorf("Send request error: %s", err)
	}

	if unsubscribeAllResponse.Error != nil {
		t.Fatalf("Unsubscribe all request error: %s", setResponse.Error.Message)
	}

}
