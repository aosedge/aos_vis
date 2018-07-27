package wsserver_test

import (
	"encoding/json"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/godbus/dbus"
	//	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"gitpct.epam.com/epmd-aepr/aos_vis/wsserver"
)

var addr string = "localhost:8088"

type visResponse struct {
	Action         string       `json:"action"`
	RequestID      string       `json:"requestId"`
	Value          *interface{} `json:"value"`
	Error          *errorInfo   `json:"error"`
	Ttl            int64        `json:"TTL"`
	SubscriptionID *string      `json:"subscriptionId"`
	Timestamp      int64        `json:"timestamp"`
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

var servConfig configuration

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
	log.Info("[TEST] GetPermission token: ", token)

	return `{"Signal.Test.RPM": "r"}`, "OK", nil
}

/*******************************************************************************
 * Main
 ******************************************************************************/

func TestMain(m *testing.M) {
	conn, err := dbus.SessionBus()
	if err != nil {
		log.Errorf("Can't create session connection: %v", err)
		os.Exit(1)
	}
	reply, err := conn.RequestName("com.aosservicemanager.vistoken",
		dbus.NameFlagDoNotQueue)
	if err != nil {
		log.Error("Can't RequestName")
		os.Exit(1)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		log.Error("Name already taken")
		os.Exit(1)
	}
	dbusserver := dbusInterface{}
	conn.Export(dbusserver, "/com/aosservicemanager/vistoken", "com.aosservicemanager.vistoken")

	file, err := os.Open("../visconfig.json")
	if err != nil {
		log.Fatal("Error opening visconfig.json: ", err)
	}

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&servConfig)
	if err != nil {
		log.Error("Error parsing visconfig.json: ", err)
		os.Exit(1)
	}

	server, err := wsserver.New(servConfig.ServerUrl, servConfig.VISCert, servConfig.VISKey)
	if err != nil {
		log.Fatal("Can't create ws server ", err)
		return
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
	u := url.URL{Scheme: "wss", Host: servConfig.ServerUrl, Path: "/"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Can't connect to ws server %v", err)
	}
	defer closeConnection(c)

	getmessage := `{"action": "get", "path": "Attribute.Vehicle.VehicleIdentification.VIN", "requestId": "8756"}`
	err = c.WriteMessage(websocket.TextMessage, []byte(getmessage))
	if err != nil {
		t.Fatalf("Can't send message to server %v", err)
	}

	_, message, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message from server %v", err)
	}

	var resp visResponse

	err = json.Unmarshal(message, &resp)
	if err != nil {
		t.Fatalf("Error parcing get response: %v", err)
	}

	if (resp.Action != "get") || (resp.RequestID != "8756") {
		t.Fatalf("Unexpected value")
	}
	if resp.Error != nil {
		t.Fatalf("Error parsing get request:  %v", err)
	}
}

func TestSet(t *testing.T) {
	log.Debug("[TEST] TestSet")

	u := url.URL{Scheme: "wss", Host: servConfig.ServerUrl, Path: "/"}
	log.Debug("[TEST] Connecting to ", u.String())

	time.Sleep(time.Second)

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Can't connect to ws server %v", err)
		return
	}
	defer c.Close()
	getmessage := `{"action": "set", "path": "Attribute.Emulator.*", "value": [ {"rectangle_long0": 100 },
	{"rectangle_lat0": 150}, {"rectangle_long1": 200 }, {"rectangle_lat1": 250} ], "requestId": "8888"}`

	err = c.WriteMessage(websocket.TextMessage, []byte(getmessage))
	if err != nil {
		t.Fatalf("Can't send message to server %v", err)
		return
	}
	_, message, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message from server %v", err)
		return
	}
	log.Debug("[TEST] read:", string(message))

	var resp visResponse
	err = json.Unmarshal(message, &resp)
	if err != nil {
		t.Fatalf("Error parcing set response: %v", err)
		return
	}

	if (resp.Action != "set") || (resp.RequestID != "8888") {
		t.Fatalf("Unexpected value")
	}
	if resp.Error != nil {
		t.Fatalf("Error parsing get request:  %v", err)
	}
}

func TestGetWithAuth(t *testing.T) {
	log.Debug("[TEST] TestGetWithAuth")

	var resp visResponse

	u := url.URL{Scheme: "wss", Host: servConfig.ServerUrl, Path: "/"}
	log.Debug("[TEST] Connecting to ", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Can't connect to ws server %v", err)
		return
	}
	defer c.Close()
	//--------------- send GET wait for error 403
	requestMsg := `{"action": "get", "path": "Signal.Test.RPM", "requestId": "8755"}`

	err = c.WriteMessage(websocket.TextMessage, []byte(requestMsg))
	if err != nil {
		t.Fatalf("Can't send message to server %v", err)
		return
	}

	_, message, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message from server: %v", err)
		return
	}
	log.Debug("[TEST] read:", string(message))

	err = json.Unmarshal(message, &resp)
	if err != nil {
		t.Fatalf("Error parsing get response: %v", err)
		return
	}

	if (resp.Action != "get") || (resp.RequestID != "8755") {
		t.Fatalf("Unexpected value")
	}

	if resp.Error == nil {
		t.Fatalf("Should be error 403")
		return
	}
	if resp.Error.Number != 403 {
		t.Fatalf("Should be error 403")
		return
	}

	//-------------- send authorize request wait OK

	requestMsg = `{"action": "authorize", "tokens" : {"authorization" : "appUID" }, "requestId": "12345"}`

	err = c.WriteMessage(websocket.TextMessage, []byte(requestMsg))
	if err != nil {
		t.Fatalf("Can't send message to server %v", err)
		return
	}
	_, message, err = c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message from server: %v", err)
		return
	}
	log.Debug("[TEST] read:", string(message))

	var resp2 visResponse
	err = json.Unmarshal(message, &resp2)
	if err != nil {
		t.Fatalf("Error parcing authorization response: %v", err)
		return
	}

	if (resp2.Action != "authorize") || (resp2.RequestID != "12345") {
		t.Fatalf("Unexpected value")
	}

	log.Debug("[TEST] read: ", resp2)
	if resp2.Error != nil {
		t.Fatalf("Error authorize %v", resp2.Error.Number)
	}

	requestMsg = `{"action": "get", "path": "Signal.Test.RPM", "requestId": "12347"}`

	err = c.WriteMessage(websocket.TextMessage, []byte(requestMsg))
	if err != nil {
		t.Fatalf("Can't send message to server %v", err)
		return
	}
	_, message, err = c.ReadMessage()
	log.Debug("[TEST] read:", string(message))
	if err != nil {
		t.Fatalf("Can't read message from server: %v", err)
		return
	}
	var resp3 visResponse
	err = json.Unmarshal(message, &resp3)
	if err != nil {
		t.Fatalf("Error parcing get response: %v", err)
		return
	}

	if (resp3.Action != "get") || (resp3.RequestID != "12347") {
		t.Fatalf("Unexpected value")
	}

	if resp3.Error != nil {
		t.Fatalf("Error get %v", resp3.Error.Number)
	}

	defer c.Close()
}

func TestSubscribeUnsubscribe(t *testing.T) {
	log.Debug("[TEST] TestSubscribeUnsubscribe")

	u := url.URL{Scheme: "wss", Host: servConfig.ServerUrl, Path: "/"}
	log.Debug("[TEST] Connecting to ", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Can't connect to ws server %v", err)
		return
	}
	defer c.Close()

	subscMessage := `{"action": "subscribe","path": "Attribute.Vehicle.VehicleIdentification.VIN", "requestId": "1004"}`

	err = c.WriteMessage(websocket.TextMessage, []byte(subscMessage))
	if err != nil {
		t.Fatalf("Can't send message to server %v", err)
		return
	}
	_, message, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message from server: %v", err)
		return
	}
	log.Debug("[TEST] read:", string(message))

	var resp visResponse
	err = json.Unmarshal(message, &resp)
	if err != nil {
		t.Fatalf("Error parcing subscribe response: %v", err)
		return
	}

	if (resp.Action != "subscribe") || (resp.RequestID != "1004") {
		t.Fatalf("Unexpected value")
	}
	if resp.Error != nil {
		t.Fatalf("Unexpected error for subscribe  %v", err)
	}
	if resp.SubscriptionID == nil {
		t.Fatalf("No subscriptionId")
	}

	unsubscMessage := `{"action": "unsubscribe", "subscriptionId": "0000", "requestId": "1004"}`

	err = c.WriteMessage(websocket.TextMessage, []byte(unsubscMessage))
	if err != nil {
		t.Fatalf("Can't send message to server %v", err)
		return
	}
	_, message2, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message from server: %v", err)
		return
	}
	log.Debug("[TEST] read:", string(message2))

	var resp2 visResponse
	err = json.Unmarshal(message2, &resp2)
	if err != nil {
		t.Fatalf("Error parcing unsubscribe response: %v", err)
		return
	}

	if (resp2.Action != "unsubscribe") || (resp2.RequestID != "1004") {
		t.Fatalf("Unexpected value")
	}
	if resp2.Error == nil {
		t.Fatalf("Unexpected positive responce ")
	}

	unsubscMessageOK := `{"action": "unsubscribe", "subscriptionId": "1", "requestId": "1004"}`

	err = c.WriteMessage(websocket.TextMessage, []byte(unsubscMessageOK))
	if err != nil {
		t.Fatalf("Can't send message to server %v", err)
	}
	_, message3, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message from server: %v", err)
	}
	log.Debug("[TEST] read:", string(message3))

	var resp3 visResponse
	err = json.Unmarshal(message3, &resp3)
	if err != nil {
		t.Fatalf("Error parcing unsubscribe response: %v", err)
	}

	if (resp3.Action != "unsubscribe") || (resp3.RequestID != "1004") {
		t.Fatalf("Unexpected value")
	}
	if resp3.Error != nil {
		t.Fatalf("Unexpected error for Unsubscribe  %v", err)
	}

	unsubscMessageAll := `{"action": "unsubscribeAll", "requestId": "1004"}`

	err = c.WriteMessage(websocket.TextMessage, []byte(unsubscMessageAll))
	if err != nil {
		t.Fatalf("Can't send message to server %v", err)
		return
	}
	_, messageAll, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message from server: %v", err)
	}
	log.Debug("[TEST] read:", string(messageAll))

	var respAll visResponse
	err = json.Unmarshal(messageAll, &respAll)
	if err != nil {
		t.Fatalf("Error parcing unsubscribe response: %v", err)
	}

	if (respAll.Action != "unsubscribeAll") || (resp3.RequestID != "1004") {
		t.Fatalf("Unexpected value")
	}
	if resp3.Error != nil {
		t.Fatalf("Unexpected error for Unsubscribe All %v", err)
	}
}
