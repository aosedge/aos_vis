package main

import (
	"bytes"
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"gitpct.epam.com/epmd-aepr/aos_vis/dataadapter"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// StorageAdapter storage adapter
type StorageAdapter struct {
	baseAdapter *dataadapter.BaseAdapter
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// NewAdapter creates adapter instance
func NewAdapter(configJSON []byte) (adapter dataadapter.DataAdapter, err error) {
	log.Info("Create storage adapter")

	localAdapter := new(StorageAdapter)

	localAdapter.baseAdapter, err = dataadapter.NewBaseAdapter()
	if err != nil {
		return nil, err
	}

	localAdapter.baseAdapter.Name = "StorageAdapter"

	var sensors struct {
		Data map[string]*dataadapter.BaseData
	}

	// Parse config
	decoder := json.NewDecoder(bytes.NewReader(configJSON))
	decoder.UseNumber()
	if err = decoder.Decode(&sensors); err != nil {
		return nil, err
	}

	localAdapter.baseAdapter.Data = sensors.Data

	return localAdapter, nil
}

// Close closes adapter
func (adapter *StorageAdapter) Close() {
	log.Info("Close storage adapter")

	adapter.baseAdapter.Close()
}

// GetName returns adapter name
func (adapter *StorageAdapter) GetName() (name string) {
	return adapter.baseAdapter.GetName()
}

// GetPathList returns list of all pathes for this adapter
func (adapter *StorageAdapter) GetPathList() (pathList []string, err error) {
	return adapter.baseAdapter.GetPathList()
}

// IsPathPublic returns true if requested data accessible without authorization
func (adapter *StorageAdapter) IsPathPublic(path string) (result bool, err error) {
	return adapter.baseAdapter.IsPathPublic(path)
}

// GetData returns data by path
func (adapter *StorageAdapter) GetData(pathList []string) (data map[string]interface{}, err error) {
	return adapter.baseAdapter.GetData(pathList)
}

// SetData sets data by pathes
func (adapter *StorageAdapter) SetData(data map[string]interface{}) (err error) {
	return adapter.baseAdapter.SetData(data)
}

// GetSubscribeChannel returns channel on which data changes will be sent
func (adapter *StorageAdapter) GetSubscribeChannel() (channel <-chan map[string]interface{}) {
	return adapter.baseAdapter.SubscribeChannel
}

// Subscribe subscribes for data changes
func (adapter *StorageAdapter) Subscribe(pathList []string) (err error) {
	return adapter.baseAdapter.Subscribe(pathList)
}

// Unsubscribe unsubscribes from data changes
func (adapter *StorageAdapter) Unsubscribe(pathList []string) (err error) {
	return adapter.baseAdapter.Unsubscribe(pathList)
}

// UnsubscribeAll unsubscribes from all data changes
func (adapter *StorageAdapter) UnsubscribeAll() (err error) {
	return adapter.baseAdapter.UnsubscribeAll()
}
