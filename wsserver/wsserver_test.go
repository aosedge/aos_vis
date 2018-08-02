package wsserver_test

import (
	"encoding/json"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/godbus/dbus"
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
	return `{"Signal.Test.RPM": "r"}`, "OK", nil
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

	file, err := os.Open("../visconfig.json")
	if err != nil {
		log.Fatal("Error opening visconfig.json: ", err)
	}

	decoder := json.NewDecoder(file)
	if err = decoder.Decode(&servConfig); err != nil {
		log.Fatalf("Error parsing visconfig.json: %s", err)
	}

	server, err := wsserver.New(servConfig.ServerUrl, servConfig.VISCert, servConfig.VISKey)
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
	u := url.URL{Scheme: "wss", Host: servConfig.ServerUrl, Path: "/"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Can't connect to ws server %s", err)
	}
	defer closeConnection(c)

	getmessage := `{"action": "get", "path": "Attribute.Vehicle.VehicleIdentification.VIN", "requestId": "8756"}`
	if err = c.WriteMessage(websocket.TextMessage, []byte(getmessage)); err != nil {
		t.Fatalf("Can't send message to server %s", err)
	}

	_, message, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message from server %s", err)
	}

	var resp visResponse

	if err = json.Unmarshal(message, &resp); err != nil {
		t.Fatalf("Error parsing get response: %s", err)
	}

	if (resp.Action != "get") || (resp.RequestID != "8756") {
		t.Fatalf("Unexpected value")
	}
	if resp.Error != nil {
		t.Fatalf("Error parsing get request:  %s", err)
	}
}

func TestSet(t *testing.T) {
	u := url.URL{Scheme: "wss", Host: servConfig.ServerUrl, Path: "/"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Can't connect to ws server %s", err)
	}
	defer closeConnection(c)

	getmessage := `{"action": "set", "path": "Attribute.Emulator.*", "value": [ {"rectangle_long0": 100 },` +
		`{"rectangle_lat0": 150}, {"rectangle_long1": 200 }, {"rectangle_lat1": 250} ], "requestId": "8888"}`

	if err = c.WriteMessage(websocket.TextMessage, []byte(getmessage)); err != nil {
		t.Fatalf("Can't send message to server %s", err)
	}
	_, message, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message from server %s", err)
	}

	var resp visResponse

	if err = json.Unmarshal(message, &resp); err != nil {
		t.Fatalf("Error parsing set response: %s", err)
	}

	if (resp.Action != "set") || (resp.RequestID != "8888") {
		t.Fatalf("Unexpected value")
	}
	if resp.Error != nil {
		t.Fatalf("Error parsing get request:  %s", err)
	}
}

func TestGetWithAuth(t *testing.T) {
	u := url.URL{Scheme: "wss", Host: servConfig.ServerUrl, Path: "/"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Can't connect to ws server %s", err)
	}
	defer closeConnection(c)

	//--------------- send GET wait for error 403

	requestMsg := `{"action": "get", "path": "Signal.Test.RPM", "requestId": "8755"}`

	if err = c.WriteMessage(websocket.TextMessage, []byte(requestMsg)); err != nil {
		t.Fatalf("Can't send message to server %s", err)
	}

	_, message, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message from server: %s", err)
	}

	var resp visResponse

	if err = json.Unmarshal(message, &resp); err != nil {
		t.Fatalf("Error parsing get response: %s", err)
	}

	if (resp.Action != "get") || (resp.RequestID != "8755") {
		t.Fatalf("Unexpected value")
	}
	if resp.Error == nil || resp.Error.Number != 401 {
		t.Fatalf("Should be error 401")
	}

	//-------------- send authorize request wait OK

	requestMsg = `{"action": "authorize", "tokens" : {"authorization" : "appUID" }, "requestId": "12345"}`

	if err = c.WriteMessage(websocket.TextMessage, []byte(requestMsg)); err != nil {
		t.Fatalf("Can't send message to server %s", err)
	}

	_, message, err = c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message from server: %s", err)
	}

	resp = visResponse{}

	if err = json.Unmarshal(message, &resp); err != nil {
		t.Fatalf("Error parsing authorization response: %s", err)
	}

	if (resp.Action != "authorize") || (resp.RequestID != "12345") {
		t.Fatalf("Unexpected value")
	}
	if resp.Error != nil {
		t.Fatalf("Error authorize: %s", resp.Error.Message)
	}

	requestMsg = `{"action": "get", "path": "Signal.Test.RPM", "requestId": "12347"}`

	if err = c.WriteMessage(websocket.TextMessage, []byte(requestMsg)); err != nil {
		t.Fatalf("Can't send message to server: %s", err)
	}

	_, message, err = c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message from server: %s", err)
	}

	resp = visResponse{}

	if err = json.Unmarshal(message, &resp); err != nil {
		t.Fatalf("Error parsing get response: %s", err)
	}

	if (resp.Action != "get") || (resp.RequestID != "12347") {
		t.Fatalf("Unexpected value")
	}

	if resp.Error != nil {
		t.Fatalf("Error get: %s", resp.Error.Message)
	}
}

func TestSubscribeUnsubscribe(t *testing.T) {
	u := url.URL{Scheme: "wss", Host: servConfig.ServerUrl, Path: "/"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Can't connect to ws server: %s", err)
	}
	defer closeConnection(c)

	subscMessage := `{"action": "subscribe","path": "Attribute.Vehicle.VehicleIdentification.VIN", "requestId": "1004"}`

	if err = c.WriteMessage(websocket.TextMessage, []byte(subscMessage)); err != nil {
		t.Fatalf("Can't send message to server: %s", err)
	}

	_, message, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message from server: %s", err)
	}

	var resp visResponse

	if err = json.Unmarshal(message, &resp); err != nil {
		t.Fatalf("Error parsing subscribe response: %s", err)
	}

	if (resp.Action != "subscribe") || (resp.RequestID != "1004") {
		t.Fatalf("Unexpected value")
	}
	if resp.Error != nil {
		t.Fatalf("Unexpected error for subscribe: %s", resp.Error.Message)
	}
	if resp.SubscriptionID == nil {
		t.Fatalf("No subscriptionId")
	}

	unsubscMessage := `{"action": "unsubscribe", "subscriptionId": "0", "requestId": "1004"}`

	if err = c.WriteMessage(websocket.TextMessage, []byte(unsubscMessage)); err != nil {
		t.Fatalf("Can't send message to server: %s", err)
	}
	_, message, err = c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message from server: %s", err)
	}

	resp = visResponse{}

	if err = json.Unmarshal(message, &resp); err != nil {
		t.Fatalf("Error parsing unsubscribe response: %s", err)
	}

	if (resp.Action != "unsubscribe") || (resp.RequestID != "1004") {
		t.Fatalf("Unexpected value")
	}
	if resp.Error == nil {
		t.Fatal("Unexpected positive response")
	}

	unsubscMessageOK := `{"action": "unsubscribe", "subscriptionId": "1", "requestId": "1004"}`

	if err = c.WriteMessage(websocket.TextMessage, []byte(unsubscMessageOK)); err != nil {
		t.Fatalf("Can't send message to server %s", err)
	}

	_, message, err = c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message from server: %s", err)
	}

	resp = visResponse{}

	if err = json.Unmarshal(message, &resp); err != nil {
		t.Fatalf("Error parsing unsubscribe response: %s", err)
	}

	if (resp.Action != "unsubscribe") || (resp.RequestID != "1004") {
		t.Fatalf("Unexpected value")
	}
	if resp.Error != nil {
		t.Fatalf("Unexpected error for unsubscribe  %s", resp.Error.Message)
	}

	unsubscMessageAll := `{"action": "unsubscribeAll", "requestId": "1004"}`

	if err = c.WriteMessage(websocket.TextMessage, []byte(unsubscMessageAll)); err != nil {
		t.Fatalf("Can't send message to server: %s", err)
	}

	_, message, err = c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message from server: %s", err)
	}

	resp = visResponse{}

	if err = json.Unmarshal(message, &resp); err != nil {
		t.Fatalf("Error parsing unsubscribe response: %s", err)
	}

	if (resp.Action != "unsubscribeAll") || (resp.RequestID != "1004") {
		t.Fatalf("Unexpected value")
	}
	if resp.Error != nil {
		t.Fatalf("Unexpected error for unsubscribe all: %s", resp.Error.Message)
	}
}
