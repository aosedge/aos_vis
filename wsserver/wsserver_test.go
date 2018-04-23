package wsserver_test

import (
	"encoding/json"
	"net/url"
	"os"
	"testing"
	//	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"gitpct.epam.com/epmd-aepr/aos_vis/wsserver"
)

var server *wsserver.WsServer
var addr string = "localhost:8088"

type visResponce struct {
	Action    string       `json:"action"`
	RequestId string       `json:"requestId"`
	Value     *interface{} `json:"value"` //TODO: redo to {}interface
	Error     *errorInfo   `json:"error"`
	Ttl       int64        `json:"TTL"`
	Timestamp int64        `json:"timestamp"`
}

type errorInfo struct {
	Number  int
	Reason  string
	Message string
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

/*******************************************************************************
 * Main
 ******************************************************************************/

func TestMain(m *testing.M) {

	server, err := wsserver.New(addr)

	if err != nil {
		log.Fatal("Can't create ws server ", err)
		return
	}

	go server.Start()

	ret := m.Run()

	server.Stop()
	os.Exit(ret)
}

func TestGetNoAuth(t *testing.T) {
	log.Debug("[TEST] TestGet")

	u := url.URL{Scheme: "ws", Host: addr, Path: "/"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Can't connect to ws server %v", err)
		return
	}
	defer c.Close()
	getmessage := `{"action": "get", "path": "Attribute.Vehicle.VehicleIdentification.VIN", "requestId": "8756"}`

	err = c.WriteMessage(websocket.TextMessage, []byte(getmessage))
	if err != nil {
		t.Fatalf("Can't send message to server ", err)
		return
	}
	_, message, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message froms erver  ", err)
		return
	}
	log.Debug("[TEST] read:", string(message))

	var resp visResponce
	err = json.Unmarshal(message, &resp)
	if err != nil {
		t.Fatalf("Error parce Get responce  ", err)
		return
	}

	if (resp.Action != "get") || (resp.RequestId != "8756") {
		t.Fatalf("Unexpected value")
	}
	if resp.Error != nil {
		t.Fatalf("Error parce Get request  ", err)
	}

}

func TestGetWithAuth(t *testing.T) {
	log.Debug("[TEST] TestGetWithAuth")

	var resp visResponce

	u := url.URL{Scheme: "ws", Host: addr, Path: "/"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Can't connect to ws server %v", err)
		return
	}
	defer c.Close()
	//--------------- send GET wait for error 403
	requestMsg := `{"action": "get", "path": "Signal.Drivetrain.InternalCombustionEngine.RPM", "requestId": "8755"}`

	err = c.WriteMessage(websocket.TextMessage, []byte(requestMsg))
	if err != nil {
		t.Fatalf("Can't send message to server ", err)
		return
	}

	_, message, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message froms erver  ", err)
		return
	}
	log.Debug("[TEST] read:", string(message))

	err = json.Unmarshal(message, &resp)
	if err != nil {
		t.Fatalf("Error parce Get responce  ", err)
		return
	}

	if (resp.Action != "get") || (resp.RequestId != "8755") {
		t.Fatalf("Unexpected value")
	}

	if resp.Error == nil {
		t.Fatalf("should be error 403")
		return
	}
	if resp.Error.Number != 403 {
		t.Fatalf("should be error 403")
		return
	}

	//-------------- send authorize request wait OK

	requestMsg = `{"action": "authorize", "tokens" : {"authorization" : "appUID" }, "requestId": "12345"}`

	err = c.WriteMessage(websocket.TextMessage, []byte(requestMsg))
	if err != nil {
		t.Fatalf("Can't send message to server ", err)
		return
	}
	_, message, err = c.ReadMessage()
	if err != nil {
		t.Fatalf("Can't read message froms erver  ", err)
		return
	}
	log.Debug("[TEST] read:", string(message))

	var resp2 visResponce
	err = json.Unmarshal(message, &resp2)
	if err != nil {
		t.Fatalf("Error parce authorization responce  ", err)
		return
	}

	if (resp2.Action != "authorize") || (resp2.RequestId != "12345") {
		t.Fatalf("Unexpected value")
	}

	log.Debug("[TEST] read: ", resp2)
	if resp2.Error != nil {
		t.Fatalf("Error authorize", resp2.Error.Number)
	}

	requestMsg = `{"action": "get", "path": "Signal.Drivetrain.InternalCombustionEngine.RPM", "requestId": "12347"}`

	err = c.WriteMessage(websocket.TextMessage, []byte(requestMsg))
	if err != nil {
		t.Fatalf("Can't send message to server ", err)
		return
	}
	_, message, err = c.ReadMessage()
	log.Debug("[TEST] read:", string(message))
	if err != nil {
		t.Fatalf("Can't read message froms erver  ", err)
		return
	}
	var resp3 visResponce
	err = json.Unmarshal(message, &resp3)
	if err != nil {
		t.Fatalf("Error parce Get responce  ", err)
		return
	}

	if (resp3.Action != "get") || (resp3.RequestId != "12347") {
		t.Fatalf("Unexpected value")
	}

	if resp3.Error != nil {
		t.Fatalf("Error get", resp3.Error.Number)
	}

	defer c.Close()
}
