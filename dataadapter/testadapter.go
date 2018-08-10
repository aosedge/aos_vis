package dataadapter

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	log "github.com/sirupsen/logrus"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// TestAdapter test adapter
type TestAdapter struct {
	data             map[string]*testData
	mutex            sync.Mutex
	subscribeChannel chan map[string]interface{}
}

type testData struct {
	subscribe bool
	value     interface{}
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// NewTestAdapter creates adapter to be used for tests
func NewTestAdapter() (adapter *TestAdapter, err error) {
	log.Info("Create test adapter")

	adapter = new(TestAdapter)

	adapter.data = make(map[string]*testData)
	adapter.subscribeChannel = make(chan map[string]interface{}, 100)

	adapter.data["Attribute.Vehicle.VehicleIdentification.VIN"] = &testData{value: "TestVIN"}
	adapter.data["Attribute.Vehicle.UserIdentification.Users"] = &testData{value: []string{"User1", "Provider1"}}

	adapter.data["Signal.Drivetrain.InternalCombustionEngine.RPM"] = &testData{value: 1000}

	adapter.data["Signal.Body.Trunk.IsLocked"] = &testData{value: false}
	adapter.data["Signal.Body.Trunk.IsOpen"] = &testData{value: true}

	adapter.data["Signal.Cabin.Door.Row1.Right.IsLocked"] = &testData{value: true}
	adapter.data["Signal.Cabin.Door.Row1.Right.Window.Position"] = &testData{value: 50}
	adapter.data["Signal.Cabin.Door.Row1.Left.IsLocked"] = &testData{value: true}
	adapter.data["Signal.Cabin.Door.Row1.Left.Window.Position"] = &testData{value: 23}
	adapter.data["Signal.Cabin.Door.Row2.Right.IsLocked"] = &testData{value: false}
	adapter.data["Signal.Cabin.Door.Row2.Right.Window.Position"] = &testData{value: 100}
	adapter.data["Signal.Cabin.Door.Row2.Left.IsLocked"] = &testData{value: true}
	adapter.data["Signal.Cabin.Door.Row2.Left.Window.Position"] = &testData{value: 0}

	return adapter, nil
}

// GetName returns adapter name
func (adapter *TestAdapter) GetName() (name string) {
	return "TestAdapter"
}

// GetPathList returns list of all pathes for this adapter
func (adapter *TestAdapter) GetPathList() (pathList []string, err error) {
	adapter.mutex.Lock()
	defer adapter.mutex.Unlock()

	pathList = make([]string, 0, len(adapter.data))

	for path := range adapter.data {
		pathList = append(pathList, path)
	}

	return pathList, nil
}

// IsPathPublic returns true if requested data accessible without authorization
func (adapter *TestAdapter) IsPathPublic(path string) (result bool, err error) {
	adapter.mutex.Lock()
	defer adapter.mutex.Unlock()

	if _, ok := adapter.data[path]; !ok {
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
	adapter.mutex.Lock()
	defer adapter.mutex.Unlock()

	data = make(map[string]interface{})

	for _, path := range pathList {
		if _, ok := adapter.data[path]; !ok {
			return data, fmt.Errorf("Path %s doesn't exits", path)
		}
		data[path] = adapter.data[path].value
	}

	return data, nil
}

// SetData sets data by pathes
func (adapter *TestAdapter) SetData(data map[string]interface{}) (err error) {
	adapter.mutex.Lock()
	defer adapter.mutex.Unlock()

	changedData := make(map[string]interface{})

	for path, value := range data {
		if _, ok := adapter.data[path]; !ok {
			return fmt.Errorf("Path %s doesn't exits", path)
		}

		oldValue := adapter.data[path].value

		if err = adapter.setData(path, value); err != nil {
			return err
		}

		if !reflect.DeepEqual(oldValue, adapter.data[path].value) &&
			adapter.data[path].subscribe {
			changedData[path] = adapter.data[path].value
		}
	}

	if len(changedData) > 0 {
		adapter.subscribeChannel <- changedData
	}

	return nil
}

// GetSubscribeChannel returns channel on which data changes will be sent
func (adapter *TestAdapter) GetSubscribeChannel() (channel <-chan map[string]interface{}) {
	return adapter.subscribeChannel
}

// Subscribe subscribes for data changes
func (adapter *TestAdapter) Subscribe(pathList []string) (err error) {
	adapter.mutex.Lock()
	defer adapter.mutex.Unlock()

	for _, path := range pathList {
		if _, ok := adapter.data[path]; !ok {
			return fmt.Errorf("Path %s doesn't exits", path)
		}

		adapter.data[path].subscribe = true
	}

	return nil
}

// Unsubscribe unsubscribes from data changes
func (adapter *TestAdapter) Unsubscribe(pathList []string) (err error) {
	adapter.mutex.Lock()
	defer adapter.mutex.Unlock()

	for _, path := range pathList {
		if _, ok := adapter.data[path]; !ok {
			return fmt.Errorf("Path %s doesn't exits", path)
		}

		adapter.data[path].subscribe = false
	}

	return nil
}

// UnsubscribeAll unsubscribes from all data changes
func (adapter *TestAdapter) UnsubscribeAll() (err error) {
	adapter.mutex.Lock()
	defer adapter.mutex.Unlock()

	for _, data := range adapter.data {
		data.subscribe = false
	}

	return nil
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func (adapter *TestAdapter) setData(path string, value interface{}) (err error) {
	switch path {
	case "Signal.Drivetrain.InternalCombustionEngine.RPM":
		return errors.New("The desired signal cannot be set since it is a read only signal")

	default:
		adapter.data[path].value = value
		return nil
	}
}
