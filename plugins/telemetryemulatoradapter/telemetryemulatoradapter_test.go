package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"gitpct.epam.com/epmd-aepr/aos_vis/dataadaptertest"
)

/*******************************************************************************
 * Var
 ******************************************************************************/

var (
	adapterInfo  dataadaptertest.TestAdapterInfo
	emulatorData map[string]interface{}
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
	startHTTPServer()

	telemetryEmulatorAdapter, err := NewAdapter([]byte(`{"SensorURL":"http://localhost:8801"}`))
	if err != nil {
		log.Fatalf("Can't create sensor emulator adapter: %s", err)
	}

	adapterInfo = dataadaptertest.TestAdapterInfo{
		Name:    "TelemetryEmulatorAdapter",
		Adapter: telemetryEmulatorAdapter,
		SetData: map[string]interface{}{
			"Attribute.Emulator.rectangle_long0": 23.56,
			"Attribute.Emulator.rectangle_lat0":  34.12,
			"Attribute.Emulator.rectangle_long1": 36.87,
			"Attribute.Emulator.rectangle_lat1":  39.21,
			"Attribute.Emulator.to_rectangle":    true},
		SubscribeList: []string{
			"Attribute.Emulator.rectangle_long0",
			"Attribute.Emulator.rectangle_lat0",
			"Attribute.Emulator.rectangle_long1",
			"Attribute.Emulator.rectangle_lat1",
			"Attribute.Emulator.to_rectangle"},
		SetSubscribeData: map[string]interface{}{
			"Attribute.Emulator.rectangle_long0": 26.56,
			"Attribute.Emulator.rectangle_lat0":  38.12,
			"Attribute.Emulator.rectangle_long1": 40.87,
			"Attribute.Emulator.rectangle_lat1":  55.21,
			"Attribute.Emulator.to_rectangle":    false}}

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

func TestGetSetData(t *testing.T) {
	if err := dataadaptertest.GetSetData(&adapterInfo); err != nil {
		t.Errorf("Test get set data error: %s", err)
	}
}

func TestSubscribeUnsubscribe(t *testing.T) {
	if err := dataadaptertest.SubscribeUnsubscribe(&adapterInfo); err != nil {
		t.Errorf("Test subscribe unsubscribe error: %s", err)
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

func startHTTPServer() {
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
