package dataadaptertest

import (
	"fmt"
	"reflect"
	"time"

	"gitpct.epam.com/epmd-aepr/aos_vis/dataadapter"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// TestAdapterInfo contains info for adapter test
type TestAdapterInfo struct {
	Adapter          dataadapter.DataAdapter
	Name             string
	PathListLen      int
	SetData          map[string]interface{}
	SetSubscribeData map[string]interface{}
	SubscribeList    []string
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// GetName tests GetName adapter method
func GetName(adapterInfo *TestAdapterInfo) (err error) {
	name := adapterInfo.Adapter.GetName()
	if name != adapterInfo.Name {
		return fmt.Errorf("Wrong adapter %s name: %s", adapterInfo.Name, name)
	}

	return nil
}

// GetPathList tests GetPathList adapter method
func GetPathList(adapterInfo *TestAdapterInfo) (err error) {
	pathList, err := adapterInfo.Adapter.GetPathList()
	if err != nil {
		return err
	}

	if adapterInfo.PathListLen != 0 && len(pathList) != adapterInfo.PathListLen {
		return fmt.Errorf("Wrong adapter %s path list len: %d", adapterInfo.Name, len(pathList))
	}

	return nil
}

// PublicPath tests IsPathPublic adapter method
func PublicPath(adapterInfo *TestAdapterInfo) (err error) {
	pathList, _ := adapterInfo.Adapter.GetPathList()
	for _, path := range pathList {
		_, err := adapterInfo.Adapter.IsPathPublic(path)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetSetData tests Get and Set adapter methods
func GetSetData(adapterInfo *TestAdapterInfo) (err error) {
	if adapterInfo.SetData == nil {
		return nil
	}

	// set data
	err = adapterInfo.Adapter.SetData(adapterInfo.SetData)
	if err != nil {
		return err
	}

	// get data
	getPathList := make([]string, 0, len(adapterInfo.SetData))
	for path := range adapterInfo.SetData {
		getPathList = append(getPathList, path)
	}
	getData, err := adapterInfo.Adapter.GetData(getPathList)
	if err != nil {
		return err
	}

	// check data
	for path, data := range getData {
		if !reflect.DeepEqual(adapterInfo.SetData[path], data) {
			return fmt.Errorf("Wrong path: %s value: %v", path, data)
		}
	}

	return nil
}

// SubscribeUnsubscribe tests Subscribe and Unsubscribe adapter methods
func SubscribeUnsubscribe(adapterInfo *TestAdapterInfo) (err error) {
	if adapterInfo.SetData == nil {
		return nil
	}

	err = adapterInfo.Adapter.SetData(adapterInfo.SetData)
	if err != nil {
		return err
	}

	// subscribe
	if err = adapterInfo.Adapter.Subscribe(adapterInfo.SubscribeList); err != nil {
		return err
	}

	if err = adapterInfo.Adapter.SetData(adapterInfo.SetSubscribeData); err != nil {
		return err
	}

	select {
	case getData := <-adapterInfo.Adapter.GetSubscribeChannel():
		// check data
		for path, data := range getData {
			if !reflect.DeepEqual(adapterInfo.SetSubscribeData[path], data) {
				return fmt.Errorf("Wrong path: %s value: %v", path, data)
			}
		}
	case <-time.After(100 * time.Millisecond):
		return fmt.Errorf("Waiting for adapter %s data timeout", adapterInfo.Name)
	}

	// unsubscribe
	if err = adapterInfo.Adapter.Unsubscribe(adapterInfo.SubscribeList); err != nil {
		return err
	}

	return nil
}
