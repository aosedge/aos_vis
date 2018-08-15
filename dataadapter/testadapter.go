package dataadapter

import (
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// TestAdapter test adapter
type TestAdapter struct {
	baseAdapter *BaseAdapter
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// NewTestAdapter creates adapter to be used for tests
func NewTestAdapter() (adapter *TestAdapter, err error) {
	log.Info("Create test adapter")

	adapter = new(TestAdapter)

	adapter.baseAdapter, err = newBaseAdapter()
	if err != nil {
		return nil, err
	}

	adapter.baseAdapter.data["Attribute.Vehicle.VehicleIdentification.VIN"] = &baseData{value: "TestVIN"}
	adapter.baseAdapter.data["Attribute.Vehicle.UserIdentification.Users"] = &baseData{value: []string{"User1", "Provider1"}}

	adapter.baseAdapter.data["Signal.Drivetrain.InternalCombustionEngine.RPM"] = &baseData{value: 1000}

	adapter.baseAdapter.data["Signal.Body.Trunk.IsLocked"] = &baseData{value: false}
	adapter.baseAdapter.data["Signal.Body.Trunk.IsOpen"] = &baseData{value: true}

	adapter.baseAdapter.data["Signal.Cabin.Door.Row1.Right.IsLocked"] = &baseData{value: true}
	adapter.baseAdapter.data["Signal.Cabin.Door.Row1.Right.Window.Position"] = &baseData{value: 50}
	adapter.baseAdapter.data["Signal.Cabin.Door.Row1.Left.IsLocked"] = &baseData{value: true}
	adapter.baseAdapter.data["Signal.Cabin.Door.Row1.Left.Window.Position"] = &baseData{value: 23}
	adapter.baseAdapter.data["Signal.Cabin.Door.Row2.Right.IsLocked"] = &baseData{value: false}
	adapter.baseAdapter.data["Signal.Cabin.Door.Row2.Right.Window.Position"] = &baseData{value: 100}
	adapter.baseAdapter.data["Signal.Cabin.Door.Row2.Left.IsLocked"] = &baseData{value: true}
	adapter.baseAdapter.data["Signal.Cabin.Door.Row2.Left.Window.Position"] = &baseData{value: 0}

	return adapter, nil
}

// GetName returns adapter name
func (adapter *TestAdapter) GetName() (name string) {
	return "TestAdapter"
}

// GetPathList returns list of all pathes for this adapter
func (adapter *TestAdapter) GetPathList() (pathList []string, err error) {
	return adapter.baseAdapter.getPathList()
}

// IsPathPublic returns true if requested data accessible without authorization
func (adapter *TestAdapter) IsPathPublic(path string) (result bool, err error) {
	adapter.baseAdapter.mutex.Lock()
	defer adapter.baseAdapter.mutex.Unlock()

	if _, ok := adapter.baseAdapter.data[path]; !ok {
		return false, fmt.Errorf("Path %s doesn't exits", path)
	}

	switch path {
	case "Attribute.Vehicle.VehicleIdentification.VIN":
		return true, nil

	case "Attribute.Vehicle.UserIdentification.Users":
		return true, nil
	}

	return false, nil
}

// GetData returns data by path
func (adapter *TestAdapter) GetData(pathList []string) (data map[string]interface{}, err error) {
	return adapter.baseAdapter.getData(pathList)
}

// SetData sets data by pathes
func (adapter *TestAdapter) SetData(data map[string]interface{}) (err error) {
	for path := range data {
		switch path {
		case "Signal.Drivetrain.InternalCombustionEngine.RPM":
			return errors.New("The desired signal cannot be set since it is a read only signal")
		}
	}

	return adapter.baseAdapter.setData(data)
}

// GetSubscribeChannel returns channel on which data changes will be sent
func (adapter *TestAdapter) GetSubscribeChannel() (channel <-chan map[string]interface{}) {
	return adapter.baseAdapter.subscribeChannel
}

// Subscribe subscribes for data changes
func (adapter *TestAdapter) Subscribe(pathList []string) (err error) {
	return adapter.baseAdapter.subscribe(pathList)
}

// Unsubscribe unsubscribes from data changes
func (adapter *TestAdapter) Unsubscribe(pathList []string) (err error) {
	return adapter.baseAdapter.unsubscribe(pathList)
}

// UnsubscribeAll unsubscribes from all data changes
func (adapter *TestAdapter) UnsubscribeAll() (err error) {
	return adapter.baseAdapter.unsubscribeAll()
}
