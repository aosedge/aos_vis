package dataadapter_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"

	"gitpct.epam.com/epmd-aepr/aos_vis/dataadapter"

	log "github.com/sirupsen/logrus"
)

var adaptersInfo []adapterData

var emulatorData map[string]interface{}

type adapterData struct {
	adapter          dataadapter.DataAdapter
	name             string
	pathListLen      int
	setData          map[string]interface{}
	setSubscribeData map[string]interface{}
	subscribeList    []string
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

	// SensorEmulatorAdapter

	startHttpServer()

	sensorEmulatorAdapter, err := dataadapter.NewSensorEmulatorAdapter([]byte(`{"SensorURL":"http://localhost:8801"}`))
	if err != nil {
		log.Fatalf("Can't create sensor emulator adapter: %s", err)
	}

	adapterInfo := adapterData{
		name:    "SensorEmulatorAdapter",
		adapter: sensorEmulatorAdapter,
		setData: map[string]interface{}{
			"Attribute.Emulator.rectangle_long0": 23.56,
			"Attribute.Emulator.rectangle_lat0":  34.12,
			"Attribute.Emulator.rectangle_long1": 36.87,
			"Attribute.Emulator.rectangle_lat1":  39.21,
			"Attribute.Emulator.to_rectangle":    true},
		subscribeList: []string{
			"Attribute.Emulator.rectangle_long0",
			"Attribute.Emulator.rectangle_lat0",
			"Attribute.Emulator.rectangle_long1",
			"Attribute.Emulator.rectangle_lat1",
			"Attribute.Emulator.to_rectangle"},
		setSubscribeData: map[string]interface{}{
			"Attribute.Emulator.rectangle_long0": 26.56,
			"Attribute.Emulator.rectangle_lat0":  38.12,
			"Attribute.Emulator.rectangle_long1": 40.87,
			"Attribute.Emulator.rectangle_lat1":  55.21,
			"Attribute.Emulator.to_rectangle":    false}}

	adaptersInfo = append(adaptersInfo, adapterInfo)

	// StorageAdapter

	configJSON := `{"Data": {
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
	}}`

	storageAdapter, err := dataadapter.NewStorageAdapter([]byte(configJSON))
	if err != nil {
		log.Fatalf("Can't create sensor emulator adapter: %s", err)
	}
	adapterInfo = adapterData{
		name:        "StorageAdapter",
		pathListLen: 13,
		adapter:     storageAdapter,
		setData: map[string]interface{}{
			"Signal.Cabin.Door.Row1.Right.IsLocked":        true,
			"Signal.Cabin.Door.Row1.Right.Window.Position": 200,
			"Signal.Cabin.Door.Row1.Left.IsLocked":         false,
			"Signal.Cabin.Door.Row1.Left.Window.Position":  100,
			"Signal.Cabin.Door.Row2.Right.IsLocked":        true,
			"Signal.Cabin.Door.Row2.Right.Window.Position": 400,
			"Signal.Cabin.Door.Row2.Left.IsLocked":         false,
			"Signal.Cabin.Door.Row2.Left.Window.Position":  50},
		subscribeList: []string{
			"Signal.Cabin.Door.Row1.Right.IsLocked",
			"Signal.Cabin.Door.Row1.Right.Window.Position",
			"Signal.Cabin.Door.Row1.Left.IsLocked",
			"Signal.Cabin.Door.Row1.Left.Window.Position",
			"Signal.Cabin.Door.Row2.Right.IsLocked",
			"Signal.Cabin.Door.Row2.Right.Window.Position",
			"Signal.Cabin.Door.Row2.Left.IsLocked",
			"Signal.Cabin.Door.Row2.Left.Window.Position"},
		setSubscribeData: map[string]interface{}{
			"Signal.Cabin.Door.Row1.Right.IsLocked":        false,
			"Signal.Cabin.Door.Row1.Right.Window.Position": 100,
			"Signal.Cabin.Door.Row1.Left.IsLocked":         true,
			"Signal.Cabin.Door.Row1.Left.Window.Position":  50,
			"Signal.Cabin.Door.Row2.Right.IsLocked":        false,
			"Signal.Cabin.Door.Row2.Right.Window.Position": 60,
			"Signal.Cabin.Door.Row2.Left.IsLocked":         true,
			"Signal.Cabin.Door.Row2.Left.Window.Position":  70},
	}

	adaptersInfo = append(adaptersInfo, adapterInfo)

	ret := m.Run()

	os.Exit(ret)
}

/*******************************************************************************
 * Tests
 ******************************************************************************/

func TestGetName(t *testing.T) {
	for _, adapterInfo := range adaptersInfo {
		name := adapterInfo.adapter.GetName()
		if name != adapterInfo.name {
			t.Errorf("Wrong adapter %s name: %s", adapterInfo.name, name)
		}
	}
}

func TestGetPathList(t *testing.T) {
	for _, adapterInfo := range adaptersInfo {
		pathList, err := adapterInfo.adapter.GetPathList()
		if err != nil {
			t.Errorf("Can't get adapter %s path list: %s", adapterInfo.name, err)
			continue
		}
		if adapterInfo.pathListLen != 0 && len(pathList) != adapterInfo.pathListLen {
			t.Errorf("Wrong adapter %s path list len: %d", adapterInfo.name, len(pathList))
		}
	}
}

func TestPublicPath(t *testing.T) {
	for _, adapterInfo := range adaptersInfo {
		pathList, _ := adapterInfo.adapter.GetPathList()
		for _, path := range pathList {
			_, err := adapterInfo.adapter.IsPathPublic(path)
			if err != nil {
				t.Errorf("Can't check adapter %s publicity: %s", adapterInfo.name, err)
			}
		}
	}
}

func TestGetSetData(t *testing.T) {
	for _, adapterInfo := range adaptersInfo {
		if adapterInfo.setData == nil {
			continue
		}

		// set data
		err := adapterInfo.adapter.SetData(adapterInfo.setData)
		if err != nil {
			t.Errorf("Can't set adapter %s data: %s", adapterInfo.name, err)
			continue
		}

		// get data
		getPathList := make([]string, 0, len(adapterInfo.setData))
		for path := range adapterInfo.setData {
			getPathList = append(getPathList, path)
		}
		getData, err := adapterInfo.adapter.GetData(getPathList)
		if err != nil {
			t.Errorf("Can't get adapter %s data: %s", adapterInfo.name, err)
			continue
		}

		// check data
		for path, data := range getData {
			if !reflect.DeepEqual(adapterInfo.setData[path], data) {
				t.Errorf("Wrong path: %s value: %v", path, data)
			}
		}
	}
}

func TestSubscribeUnsubscribe(t *testing.T) {
	for _, adapterInfo := range adaptersInfo {
		if adapterInfo.setData == nil {
			continue
		}

		err := adapterInfo.adapter.SetData(adapterInfo.setData)
		if err != nil {
			t.Errorf("Can't set adapter %s data: %s", adapterInfo.name, err)
			continue
		}

		// subscribe
		if err = adapterInfo.adapter.Subscribe(adapterInfo.subscribeList); err != nil {
			t.Errorf("Can't subscribe adapter %s path: %s", adapterInfo.name, err)
			continue
		}

		if err = adapterInfo.adapter.SetData(adapterInfo.setSubscribeData); err != nil {
			t.Errorf("Can't set adapter %s data: %s", adapterInfo.name, err)
			continue
		}

		select {
		case getData := <-adapterInfo.adapter.GetSubscribeChannel():
			// check data
			for path, data := range getData {
				if !reflect.DeepEqual(adapterInfo.setSubscribeData[path], data) {
					t.Errorf("Wrong path: %s value: %v", path, data)
				}
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Waiting for adapter %s data timeout", adapterInfo.name)
		}
	}
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func statsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	dataJSON, err := json.Marshal(emulatorData)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	w.Write(dataJSON)
}

func attributesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	dataJSON, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	if err = json.Unmarshal(dataJSON, &emulatorData); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func startHttpServer() {
	emulatorData = map[string]interface{}{
		"rectangle_lat0":  nil,
		"rectangle_lat1":  nil,
		"rectangle_long0": nil,
		"rectangle_long1": nil,
		"to_rectangle":    nil}

	http.HandleFunc("/stats/", statsHandler)
	http.HandleFunc("/attributes/", attributesHandler)
	go http.ListenAndServe("localhost:8801", nil)

	time.Sleep(1 * time.Second)
}
