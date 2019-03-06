package dataadapter

import (
	"fmt"
	"reflect"
	"sync"
)

/*******************************************************************************
 * Consts
 ******************************************************************************/

const subscribeChannelSize = 32

/*******************************************************************************
 * Types
 ******************************************************************************/

// BaseAdapter base adapter
type BaseAdapter struct {
	Name             string
	Data             map[string]*BaseData
	Mutex            sync.Mutex
	SubscribeChannel chan map[string]interface{}
}

// BaseData base data type
type BaseData struct {
	Public    bool
	ReadOnly  bool
	Subscribe bool
	Value     interface{}
}

/*******************************************************************************
 * Private
 ******************************************************************************/

// NewBaseAdapter creates base adapter
func NewBaseAdapter() (adapter *BaseAdapter, err error) {
	adapter = new(BaseAdapter)

	adapter.Data = make(map[string]*BaseData)
	adapter.SubscribeChannel = make(chan map[string]interface{}, subscribeChannelSize)

	return adapter, nil
}

// Close closes adapter
func (adapter *BaseAdapter) Close() {
}

// GetName returns adapter name
func (adapter *BaseAdapter) GetName() (name string) {
	return adapter.Name
}

// GetPathList returns list of all pathes for this adapter
func (adapter *BaseAdapter) GetPathList() (pathList []string, err error) {
	adapter.Mutex.Lock()
	defer adapter.Mutex.Unlock()

	pathList = make([]string, 0, len(adapter.Data))

	for path := range adapter.Data {
		pathList = append(pathList, path)
	}

	return pathList, nil
}

// IsPathPublic returns true if requested data accessible without authorization
func (adapter *BaseAdapter) IsPathPublic(path string) (result bool, err error) {
	adapter.Mutex.Lock()
	defer adapter.Mutex.Unlock()

	if _, ok := adapter.Data[path]; !ok {
		return result, fmt.Errorf("Path %s doesn't exits", path)
	}

	return adapter.Data[path].Public, nil
}

// GetData returns data by path
func (adapter *BaseAdapter) GetData(pathList []string) (data map[string]interface{}, err error) {
	adapter.Mutex.Lock()
	defer adapter.Mutex.Unlock()

	data = make(map[string]interface{})

	for _, path := range pathList {
		if _, ok := adapter.Data[path]; !ok {
			return data, fmt.Errorf("Path %s doesn't exits", path)
		}
		data[path] = adapter.Data[path].Value
	}

	return data, nil
}

// SetData sets data by pathes
func (adapter *BaseAdapter) SetData(data map[string]interface{}) (err error) {
	adapter.Mutex.Lock()
	defer adapter.Mutex.Unlock()

	changedData := make(map[string]interface{})

	for path, value := range data {
		if _, ok := adapter.Data[path]; !ok {
			return fmt.Errorf("Path %s doesn't exits", path)
		}

		if adapter.Data[path].ReadOnly {
			return fmt.Errorf("Signal %s cannot be set since it is a read only signal", path)
		}

		oldValue := adapter.Data[path].Value
		adapter.Data[path].Value = value

		if !reflect.DeepEqual(oldValue, value) && adapter.Data[path].Subscribe {
			changedData[path] = value
		}
	}

	if len(changedData) > 0 {
		adapter.SubscribeChannel <- changedData
	}

	return nil
}

// GetSubscribeChannel returns channel on which data changes will be sent
func (adapter *BaseAdapter) GetSubscribeChannel() (channel <-chan map[string]interface{}) {
	return adapter.SubscribeChannel
}

// Subscribe subscribes for data changes
func (adapter *BaseAdapter) Subscribe(pathList []string) (err error) {
	adapter.Mutex.Lock()
	defer adapter.Mutex.Unlock()

	for _, path := range pathList {
		if _, ok := adapter.Data[path]; !ok {
			return fmt.Errorf("Path %s doesn't exits", path)
		}

		adapter.Data[path].Subscribe = true
	}

	return nil
}

// Unsubscribe unsubscribes from data changes
func (adapter *BaseAdapter) Unsubscribe(pathList []string) (err error) {
	adapter.Mutex.Lock()
	defer adapter.Mutex.Unlock()

	for _, path := range pathList {
		if _, ok := adapter.Data[path]; !ok {
			return fmt.Errorf("Path %s doesn't exits", path)
		}

		adapter.Data[path].Subscribe = false
	}

	return nil
}

// UnsubscribeAll unsubscribes from all data changes
func (adapter *BaseAdapter) UnsubscribeAll() (err error) {
	adapter.Mutex.Lock()
	defer adapter.Mutex.Unlock()

	for _, data := range adapter.Data {
		data.Subscribe = false
	}

	return nil
}
