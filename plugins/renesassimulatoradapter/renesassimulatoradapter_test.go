package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"gitpct.epam.com/epmd-aepr/aos_vis/dataadaptertest"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

type messageType struct {
	Cmd string `json:"cmd"`
	Arg struct {
		Geometry struct {
			Coordinates struct {
				Latitude  float64
				Longitude float64
			} `json:"coordinates"`
		} `json:"geometry"`
		RunningStatus struct {
			Vehicle struct {
				Speed float64
			}
			Fuel struct {
				Level float64
			}
		}
		Body struct {
			Door struct {
				FrontLeft struct {
					IsOpen         bool
					IsLocked       bool
					WindowPosition float64
					IsMirrorOpen   bool
				}
			}
			Trunk struct {
				IsOpen bool
			}
		}
	} `json:"arg"`
}

/*******************************************************************************
 * Var
 ******************************************************************************/

var (
	adapterInfo dataadaptertest.TestAdapterInfo
	connection  *websocket.Conn
)

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
	renesasSimulatorAdapter, err := NewAdapter([]byte(`
{
	"ServerURL": ":9000",
	"Signals": {
		"geometry.coordinates.Latitude":           "Signal.Cabin.Infotainment.Navigation.CurrentLocation.Latitude",
		"geometry.coordinates.Longitude":          "Signal.Cabin.Infotainment.Navigation.CurrentLocation.Longitude",
		"RunningStatus.Vehicle.Speed":             "Signal.Vehicle.Speed",
		"RunningStatus.Fuel.Level":                "Signal.Drivetrain.FuelSystem.Level",
		"Body.Door.FrontLeft.IsOpen":              "Signal.Cabin.Door.Row1.Left.IsOpen",
		"Body.Door.FrontLeft.IsLocked":            "Signal.Cabin.Door.Row1.Left.IsLocked",
		"Body.Door.FrontLeft.WindowPosition":      "Signal.Cabin.Door.Row1.Left.Window.Position",
		"Body.Door.FrontLeft.IsMirrorOpen":        "Signal.Body.Mirrors.Left.Pan",
		"Body.Trunk.IsOpen":                       "Signal.Body.Trunk.IsOpen"
	}
}`))
	if err != nil {
		log.Fatalf("Can't create Renesas simulator adapter: %s", err)
	}
	defer renesasSimulatorAdapter.Close()

	adapterInfo = dataadaptertest.TestAdapterInfo{
		Name:    "RenesasSimulatorAdapter",
		Adapter: renesasSimulatorAdapter,
		SubscribeList: []string{
			"Signal.Cabin.Infotainment.Navigation.CurrentLocation.Latitude",
			"Signal.Cabin.Infotainment.Navigation.CurrentLocation.Longitude",
			"Signal.Vehicle.Speed",
			"Signal.Drivetrain.FuelSystem.Level",
			"Signal.Cabin.Door.Row1.Left.IsOpen",
			"Signal.Cabin.Door.Row1.Left.IsLocked",
			"Signal.Cabin.Door.Row1.Left.Window.Position",
			"Signal.Body.Mirrors.Left.Pan",
			"Signal.Body.Trunk.IsOpen"}}

	time.Sleep(1 * time.Second)

	connection, _, err = websocket.DefaultDialer.Dial("ws://localhost:9000", nil)
	if err != nil {
		log.Fatalf("Can't connect to simulator adapter: %s", err)
	}
	defer connection.Close()

	ret := m.Run()

	os.Exit(ret)
}

/*******************************************************************************
 * Tests
 ******************************************************************************/

func TestGetName(t *testing.T) {
	if err := dataadaptertest.GetName(&adapterInfo); err != nil {
		t.Errorf("Test get name error: %s", err)
	}
}

func TestGetPathList(t *testing.T) {
	if err := dataadaptertest.GetPathList(&adapterInfo); err != nil {
		t.Errorf("Test get path lis error: %s", err)
	}
}

func TestPublicPath(t *testing.T) {
	if err := dataadaptertest.PublicPath(&adapterInfo); err != nil {
		t.Errorf("Test public path error: %s", err)
	}
}

func TestGetData(t *testing.T) {
	message := messageType{Cmd: "data"}
	message.Arg.Geometry.Coordinates.Latitude = 75.34455
	message.Arg.Geometry.Coordinates.Longitude = 34.56654
	message.Arg.RunningStatus.Vehicle.Speed = 120.23
	message.Arg.RunningStatus.Fuel.Level = 50.3
	message.Arg.Body.Door.FrontLeft.IsOpen = false
	message.Arg.Body.Door.FrontLeft.IsLocked = true
	message.Arg.Body.Door.FrontLeft.WindowPosition = 23
	message.Arg.Body.Door.FrontLeft.IsMirrorOpen = false
	message.Arg.Body.Trunk.IsOpen = false

	dataMap := make(map[string]interface{})

	dataMap["Signal.Cabin.Infotainment.Navigation.CurrentLocation.Latitude"] = message.Arg.Geometry.Coordinates.Latitude
	dataMap["Signal.Cabin.Infotainment.Navigation.CurrentLocation.Longitude"] = message.Arg.Geometry.Coordinates.Longitude
	dataMap["Signal.Vehicle.Speed"] = message.Arg.RunningStatus.Vehicle.Speed
	dataMap["Signal.Drivetrain.FuelSystem.Level"] = message.Arg.RunningStatus.Fuel.Level
	dataMap["Signal.Cabin.Door.Row1.Left.IsOpen"] = message.Arg.Body.Door.FrontLeft.IsOpen
	dataMap["Signal.Cabin.Door.Row1.Left.IsLocked"] = message.Arg.Body.Door.FrontLeft.IsLocked
	dataMap["Signal.Cabin.Door.Row1.Left.Window.Position"] = message.Arg.Body.Door.FrontLeft.WindowPosition
	dataMap["Signal.Body.Mirrors.Left.Pan"] = message.Arg.Body.Door.FrontLeft.IsMirrorOpen
	dataMap["Signal.Body.Trunk.IsOpen"] = message.Arg.Body.Trunk.IsOpen

	jsonData, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("Can't marshal message: %s", err)
	}

	if err := connection.WriteMessage(websocket.TextMessage, jsonData); err != nil {
		t.Fatalf("Can't write websocket message: %s", err)
	}

	time.Sleep(1 * time.Second)

	dataPath := make([]string, 0, len(dataMap))

	for path := range dataMap {
		dataPath = append(dataPath, path)
	}

	data, err := adapterInfo.Adapter.GetData(dataPath)
	if err != nil {
		t.Fatalf("Can't get adapter data: %s", err)
	}

	for path, value := range data {
		if value != dataMap[path] {
			t.Errorf("Wrong value: path %s, value %v", path, value)
		}
	}
}

func TestSubscribeUnsubscribe(t *testing.T) {
	if err := adapterInfo.Adapter.Subscribe(adapterInfo.SubscribeList); err != nil {
		t.Fatalf("Can't write websocket message: %s", err)
	}

	message := messageType{Cmd: "data"}
	message.Arg.Geometry.Coordinates.Latitude = 71.96856
	message.Arg.Geometry.Coordinates.Longitude = 25.06834
	message.Arg.RunningStatus.Vehicle.Speed = 99.99
	message.Arg.RunningStatus.Fuel.Level = 32.23
	message.Arg.Body.Door.FrontLeft.IsOpen = true
	message.Arg.Body.Door.FrontLeft.IsLocked = false
	message.Arg.Body.Door.FrontLeft.WindowPosition = 59
	message.Arg.Body.Door.FrontLeft.IsMirrorOpen = true
	message.Arg.Body.Trunk.IsOpen = true

	dataMap := make(map[string]interface{})

	dataMap["Signal.Cabin.Infotainment.Navigation.CurrentLocation.Latitude"] = message.Arg.Geometry.Coordinates.Latitude
	dataMap["Signal.Cabin.Infotainment.Navigation.CurrentLocation.Longitude"] = message.Arg.Geometry.Coordinates.Longitude
	dataMap["Signal.Vehicle.Speed"] = message.Arg.RunningStatus.Vehicle.Speed
	dataMap["Signal.Drivetrain.FuelSystem.Level"] = message.Arg.RunningStatus.Fuel.Level
	dataMap["Signal.Cabin.Door.Row1.Left.IsOpen"] = message.Arg.Body.Door.FrontLeft.IsOpen
	dataMap["Signal.Cabin.Door.Row1.Left.IsLocked"] = message.Arg.Body.Door.FrontLeft.IsLocked
	dataMap["Signal.Cabin.Door.Row1.Left.Window.Position"] = message.Arg.Body.Door.FrontLeft.WindowPosition
	dataMap["Signal.Body.Mirrors.Left.Pan"] = message.Arg.Body.Door.FrontLeft.IsMirrorOpen
	dataMap["Signal.Body.Trunk.IsOpen"] = message.Arg.Body.Trunk.IsOpen

	jsonData, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("Can't marshal message: %s", err)
	}

	if err := connection.WriteMessage(websocket.TextMessage, jsonData); err != nil {
		t.Fatalf("Can't write websocket message: %s", err)
	}

	select {
	case data := <-adapterInfo.Adapter.GetSubscribeChannel():
		for path, value := range data {
			if value != dataMap[path] {
				t.Errorf("Wrong value: path %s, value %v", path, value)
			}
		}

	case <-time.After(100 * time.Millisecond):
		t.Errorf("Waiting for adapter %s data timeout", adapterInfo.Name)
	}

	if err = adapterInfo.Adapter.Unsubscribe(adapterInfo.SubscribeList); err != nil {
		t.Errorf("Can't unsubscribe from adapter: %s", err)
	}
}
