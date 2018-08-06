package dataadapter

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// TestAdapter test adapter
type TestAdapter struct {
	data map[string]interface{}
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// NewTestAdapter creates adapter to be used for tests
func NewTestAdapter() (adapter *TestAdapter, err error) {
	log.Debug("Create test adapter")

	adapter = new(TestAdapter)

	adapter.data = make(map[string]interface{})

	adapter.data["Attribute.Vehicle.VehicleIdentification.VIN"] = "TestVIN"
	adapter.data["Attribute.Vehicle.UserIdentification.Users"] = []string{"User1", "Provider1"}
	adapter.data["Sensor.Engine.RPM"] = 1000

	return adapter, nil
}

// GetName returns adapter name
func (adapter *TestAdapter) GetName() (name string) {
	return "TestAdapter"
}

// GetPathList returns list of all pathes for this adapter
func (adapter *TestAdapter) GetPathList() (pathList []string, err error) {
	pathList = make([]string, 0, len(adapter.data))

	for path := range adapter.data {
		pathList = append(pathList, path)
	}

	return pathList, nil
}

// IsPathPublic returns true if requested data accessible without authorization
func (adapter *TestAdapter) IsPathPublic(path string) (result bool, err error) {
	if _, ok := adapter.data[path]; !ok {
		return false, fmt.Errorf("Path %s doesn't exits", path)
	}

	switch path {
	case "Attribute.Vehicle.VehicleIdentification.VIN":
		return true, nil

	case "Attribute.Vehicle.UserIdentification.Users":
		return true, nil

	case "Sensor.Engine.RPM":
		return false, nil
	}

	return false, nil
}

// GetData returns data by path
func (adapter *TestAdapter) GetData(pathList []string) (data map[string]interface{}, err error) {
	return
}

// SetData sets data by pathes
func (adapter *TestAdapter) SetData(data map[string]interface{}) (err error) {
	return
}
