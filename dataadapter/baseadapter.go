package dataadapter

import (
	"fmt"
	"reflect"
	"sync"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// BaseAdapter test adapter
type BaseAdapter struct {
	name             string
	data             map[string]*baseData
	mutex            sync.Mutex
	subscribeChannel chan map[string]interface{}
}

type baseData struct {
	Public    bool
	ReadOnly  bool
	subscribe bool
	Value     interface{}
}

/*******************************************************************************
 * Private
 ******************************************************************************/

// newBaseAdapter creates base adapter
func newBaseAdapter() (adapter *BaseAdapter, err error) {
	adapter = new(BaseAdapter)

	adapter.data = make(map[string]*baseData)
	adapter.subscribeChannel = make(chan map[string]interface{}, 100)

	return adapter, nil
}

// GetName returns adapter name
func (adapter *BaseAdapter) getName() (name string) {
	return adapter.name
}

// getPathList returns list of all pathes for this adapter
func (adapter *BaseAdapter) getPathList() (pathList []string, err error) {
	adapter.mutex.Lock()
	defer adapter.mutex.Unlock()

	pathList = make([]string, 0, len(adapter.data))

	for path := range adapter.data {
		pathList = append(pathList, path)
	}

	return pathList, nil
}

func (adapter *BaseAdapter) isPathPublic(path string) (result bool, err error) {
	adapter.mutex.Lock()
	defer adapter.mutex.Unlock()

	if _, ok := adapter.data[path]; !ok {
		return result, fmt.Errorf("Path %s doesn't exits", path)
	}

	return adapter.data[path].Public, nil
}

// getData returns data by path
func (adapter *BaseAdapter) getData(pathList []string) (data map[string]interface{}, err error) {
	adapter.mutex.Lock()
	defer adapter.mutex.Unlock()

	data = make(map[string]interface{})

	for _, path := range pathList {
		if _, ok := adapter.data[path]; !ok {
			return data, fmt.Errorf("Path %s doesn't exits", path)
		}
		data[path] = adapter.data[path].Value
	}

	return data, nil
}

// setData sets data by pathes
func (adapter *BaseAdapter) setData(data map[string]interface{}) (err error) {
	adapter.mutex.Lock()
	defer adapter.mutex.Unlock()

	changedData := make(map[string]interface{})

	for path, value := range data {
		if _, ok := adapter.data[path]; !ok {
			return fmt.Errorf("Path %s doesn't exits", path)
		}

		if adapter.data[path].ReadOnly {
			return fmt.Errorf("Signal %s cannot be set since it is a read only signal", path)
		}

		oldValue := adapter.data[path].Value
		adapter.data[path].Value = value

		if !reflect.DeepEqual(oldValue, value) && adapter.data[path].subscribe {
			changedData[path] = value
		}
	}

	if len(changedData) > 0 {
		adapter.subscribeChannel <- changedData
	}

	return nil
}

// subscribe subscribes for data changes
func (adapter *BaseAdapter) subscribe(pathList []string) (err error) {
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

// unsubscribe unsubscribes from data changes
func (adapter *BaseAdapter) unsubscribe(pathList []string) (err error) {
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

// unsubscribeAll unsubscribes from all data changes
func (adapter *BaseAdapter) unsubscribeAll() (err error) {
	adapter.mutex.Lock()
	defer adapter.mutex.Unlock()

	for _, data := range adapter.data {
		data.subscribe = false
	}

	return nil
}
