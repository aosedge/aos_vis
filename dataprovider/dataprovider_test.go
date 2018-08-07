package dataprovider_test

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"gitpct.epam.com/epmd-aepr/aos_vis/config"
	"gitpct.epam.com/epmd-aepr/aos_vis/dataprovider"
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
 * Tests
 ******************************************************************************/

func TestPublicPath(t *testing.T) {
	provider, err := dataprovider.New(&config.Config{})
	if err != nil {
		t.Fatalf("Can't create data provider: %s", err)
	}

	path := "Attribute.Vehicle.VehicleIdentification.VIN"
	result, err := provider.IsPathPublic(path)
	if err != nil {
		t.Fatalf("Can't check path publicity: %s", err)
	}
	if !result {
		t.Errorf("Path %s should be public", path)
	}

	path = "Signal.Drivetrain.InternalCombustionEngine.RPM"
	result, err = provider.IsPathPublic(path)
	if err != nil {
		t.Fatalf("Can't check path publicity: %s", err)
	}
	if result {
		t.Errorf("Path %s should not be public", path)
	}
}

func TestGetData(t *testing.T) {
	provider, err := dataprovider.New(&config.Config{})
	if err != nil {
		t.Fatalf("Can't create data provider: %s", err)
	}

	/*
		client -> {
			"action": "get",
			"path": "Signal.Drivetrain.InternalCombustionEngine.RPM",
			"requestId": "8756"
		}

		receive <- {
			"action": "get",
			"requestId": "8756",
			"value": 2372,
			"timestamp": 1489985044000
		}
	*/

	data, err := provider.GetData("Signal.Drivetrain.InternalCombustionEngine.RPM")
	if err != nil {
		t.Errorf("Can't get data: %s", err)
	}
	if _, ok := data.(int); !ok {
		t.Errorf("Wrong data type: %s", reflect.TypeOf(data))
	}

	/*
		client -> {
			"action": "get",
			"path": "Signal.Body.Trunk",
			"requestId": "9078"
		}

		receive <- {
			"action": "get",
			"requestId": "9078",
			"value": { "Signal.Body.Trunk.IsLocked": false,
				"Signal.Body.Trunk.IsOpen": true },
			"timestamp": 1489985044000
		}
	*/

	data, err = provider.GetData("Signal.Body.Trunk")
	if err != nil {
		t.Errorf("Can't get data: %s", err)
	}
	if value, ok := data.(map[string]interface{}); !ok {
		t.Errorf("Wrong data type: %s", reflect.TypeOf(data))
	} else {
		if len(value) != 2 {
			t.Errorf("Wrong map size: %d", len(value))
		}
	}

	/*
		client -> {
			"action": "get",
			"path": "Signal.Cabin.Door.*.IsLocked",
			"requestId": "4523"
		}

		receive <- {
			"action": "get",
			"requestId": "4523",
			"value": [ {"Signal.Cabin.Door.Row1.Right.IsLocked" : true },
			           {"Signal.Cabin.Door.Row1.Left.IsLocked" : true },
				       {"Signal.Cabin.Door.Row2.Right.IsLocked" : false },
				       {"Signal.Cabin.Door.Row2.Left.IsLocked" : true } ],
			"timestamp": 1489985044000
		}
	*/

	data, err = provider.GetData("Signal.Cabin.Door.*.IsLocked")
	if err != nil {
		t.Errorf("Can't get data: %s", err)
	}
	if value, ok := data.([]map[string]interface{}); !ok {
		t.Errorf("Wrong data type: %s", reflect.TypeOf(data))
	} else {
		if len(value) != 4 {
			t.Errorf("Wrong array size: %d", len(value))
		}
		for _, item := range value {
			if len(item) != 1 {
				t.Errorf("Wrong map size: %d", len(item))
			}
		}
	}

	/*
		client -> {
			"action": "get",
			"path": "Signal.Cabin.Door.*",
			"requestId": "6745"
		}

		receive <- {
			"action": "get",
			"requestId": "6745",
			"value": [ {"Signal.Cabin.Door.Row1.Right.IsLocked" : true, "Signal.Cabin.Door.Row1.Right.Window.Position": 50},
			           {"Signal.Cabin.Door.Row1.Left.IsLocked" : true, "Signal.Cabin.Door.Row1.Left.Window.Position": 23},
			           {"Signal.Cabin.Door.Row2.Right.IsLocked" : false, "Signal.Cabin.Door.Row2.Right.Window.Position": 100 },
			           {"Signal.Cabin.Door.Row2.Left.IsLocked": true, "Signal.Cabin.Door.Row2.Left.Window.Position": 0 } ],
			"timestamp": 1489985044000
		}
	*/

	/*
		TODO: This case is unclear as Window.Position is subnode of Door.Row1.Right. We should define
		how to combine results in one map.
	*/

	data, err = provider.GetData("Signal.Cabin.Door.*")
	if err != nil {
		t.Errorf("Can't get data: %s", err)
	}
	if value, ok := data.([]map[string]interface{}); !ok {
		t.Errorf("Wrong data type: %s", reflect.TypeOf(data))
	} else {
		if len(value) != 8 {
			t.Errorf("Wrong array size: %d", len(value))
		}
		for _, item := range value {
			if len(item) != 1 {
				t.Errorf("Wrong map size: %d", len(item))
			}
		}
	}

	/*
		client -> {
			"action": "get",
			"path": "Body.Flux.Capacitor",
			"requestId": "1245"
		}

		receive <- {
			"action": "get",
			"requestId": "1245",
			"error": { "number":404,
				"reason": "invalid_path",
				"message": "The specified data path does not exist." },
			"timestamp": 1489985044000
		}
	*/

	data, err = provider.GetData("Body.Flux.Capacitor")
	if err == nil {
		t.Error("Path should not exists")
	} else if !strings.Contains(err.Error(), "not exist") {
		t.Errorf("Wrong error type: %s", err)
	}
}

func TestSetData(t *testing.T) {
	provider, err := dataprovider.New(&config.Config{})
	if err != nil {
		t.Fatalf("Can't create data provider: %s", err)
	}

	// Set by full path

	if err = provider.SetData("Signal.Body.Trunk.IsLocked", true); err != nil {
		t.Errorf("Can't set data: %s", err)
	}
	value, err := provider.GetData("Signal.Body.Trunk.IsLocked")
	if err != nil {
		t.Errorf("Can't get data: %s", err)
	}
	if value != true {
		t.Errorf("Data mistmatch: %v", value)
	}

	/*
		client -> {
			"action": "set",
			"path": "Signal.Cabin.Door.*.IsLocked",
			"value": [ {"Row1.Right.IsLocked": true },
			           {"Row1.Left.IsLocked": true },
			           {"Row2.Right.IsLocked": true },
			           {"Row2.Left.IsLocked": true } ],
			"requestId": "5689"
		}

		receive <- {
			"action": "set",
			"requestId": "5689",
			"timestamp": 1489985044000
		}
	*/

	if err = provider.SetData("Signal.Cabin.Door.*.IsLocked", []map[string]interface{}{
		{"Row1.Right.IsLocked": true},
		{"Row1.Left.IsLocked": true},
		{"Row2.Right.IsLocked": true},
		{"Row2.Left.IsLocked": true}}); err != nil {
		t.Errorf("Can't set data: %s", err)
	}
	if value, err = provider.GetData("Signal.Cabin.Door.*.IsLocked"); err != nil {
		t.Errorf("Can't get data: %s", err)
	}
	dataMap, err := arrayToMap(value)
	if err != nil {
		t.Error(err)
	}
	if dataMap["Signal.Cabin.Door.Row1.Right.IsLocked"] != true {
		t.Errorf("Data mistmatch: %v", value)
	}
	if dataMap["Signal.Cabin.Door.Row1.Left.IsLocked"] != true {
		t.Errorf("Data mistmatch: %v", value)
	}
	if dataMap["Signal.Cabin.Door.Row2.Right.IsLocked"] != true {
		t.Errorf("Data mistmatch: %v", value)
	}
	if dataMap["Signal.Cabin.Door.Row2.Left.IsLocked"] != true {
		t.Errorf("Data mistmatch: %v", value)
	}

	/*
		client -> {
			"action": "set",
			"path": "Signal.Drivetrain.InternalCombustionEngine.RPM",
			"value": 2000,
			"requestId": "8912"
		}

		receive <- {
			"action": "set",
			"requestId": "8912",
			"error": { "number": 401,
			"reason": "read_only",
			"message": "The desired signal cannot be set since it is a read only signal"},
			"timestamp": 1489985044000
		}
	*/

	err = provider.SetData("Signal.Drivetrain.InternalCombustionEngine.RPM", 2000)
	if err == nil {
		t.Error("Path should be read only")
	} else if !strings.Contains(err.Error(), "read only") {
		t.Errorf("Wrong error type: %s", err)
	}

	/*
		client -> {
			"action": "set",
			"path": "Signal.Drivetrain.InternalCombustionEngine.RPM",
			"value": { "locked" : true }
			"requestId": "2311"
			}

		receive <- {
			"action": "set",
			"requestId": "2311",
			"error": { "number": 400,
			"reason": "bad_request" ,
			"message": "The server is unable to fulfil the client request because the request is malformed."},
			"timestamp": 1489985044000
		}
	*/

	err = provider.SetData("Signal.Drivetrain.InternalCombustionEngine.RPM", map[string]interface{}{"locked": true})
	if err == nil {
		t.Error("Path should be read only")
	} else if !strings.Contains(err.Error(), "malformed") {
		t.Errorf("Wrong error type: %s", err)
	}
}

/*******************************************************************************
 * Private
 ******************************************************************************/
func arrayToMap(data interface{}) (result map[string]interface{}, err error) {
	// Create map from array
	array, ok := data.([]map[string]interface{})
	if !ok {
		return result, fmt.Errorf("Wrong data type: %s", reflect.TypeOf(data))
	}

	result = make(map[string]interface{})
	for _, arrayItem := range array {
		for path, value := range arrayItem {
			result[path] = value
		}
	}

	return result, nil
}
