package dataprovider_test

import (
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
		t.Fatalf("Can't get data: %s", err)
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
		t.Fatalf("Can't get data: %s", err)
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
		t.Fatalf("Can't get data: %s", err)
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
		t.Fatalf("Can't get data: %s", err)
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
