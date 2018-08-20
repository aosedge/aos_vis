package dataadapter

import (
	"bytes"
	"encoding/json"

	log "github.com/sirupsen/logrus"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// StorageAdapter storage adapter
type StorageAdapter struct {
	baseAdapter *BaseAdapter
}

/*******************************************************************************
c * Public
 ******************************************************************************/

// NewStorageAdapter creates adapter which store values only
func NewStorageAdapter(configJSON []byte) (adapter *StorageAdapter, err error) {
	log.Info("Create storage adapter")

	adapter = new(StorageAdapter)

	adapter.baseAdapter, err = newBaseAdapter()
	if err != nil {
		return nil, err
	}

	adapter.baseAdapter.name = "StorageAdapter"

	var sensors struct{ Data map[string]*baseData }

	// Parse config
	decoder := json.NewDecoder(bytes.NewReader(configJSON))
	decoder.UseNumber()
	if err = decoder.Decode(&sensors); err != nil {
		return nil, err
	}

	adapter.baseAdapter.data = sensors.Data

	return adapter, nil
}

// GetName returns adapter name
func (adapter *StorageAdapter) GetName() (name string) {
	return adapter.baseAdapter.getName()
}

// GetPathList returns list of all pathes for this adapter
func (adapter *StorageAdapter) GetPathList() (pathList []string, err error) {
	return adapter.baseAdapter.getPathList()
}

// IsPathPublic returns true if requested data accessible without authorization
func (adapter *StorageAdapter) IsPathPublic(path string) (result bool, err error) {
	return adapter.baseAdapter.isPathPublic(path)
}

// GetData returns data by path
func (adapter *StorageAdapter) GetData(pathList []string) (data map[string]interface{}, err error) {
	return adapter.baseAdapter.getData(pathList)
}

// SetData sets data by pathes
func (adapter *StorageAdapter) SetData(data map[string]interface{}) (err error) {
	return adapter.baseAdapter.setData(data)
}

// GetSubscribeChannel returns channel on which data changes will be sent
func (adapter *StorageAdapter) GetSubscribeChannel() (channel <-chan map[string]interface{}) {
	return adapter.baseAdapter.subscribeChannel
}

// Subscribe subscribes for data changes
func (adapter *StorageAdapter) Subscribe(pathList []string) (err error) {
	return adapter.baseAdapter.subscribe(pathList)
}

// Unsubscribe unsubscribes from data changes
func (adapter *StorageAdapter) Unsubscribe(pathList []string) (err error) {
	return adapter.baseAdapter.unsubscribe(pathList)
}

// UnsubscribeAll unsubscribes from all data changes
func (adapter *StorageAdapter) UnsubscribeAll() (err error) {
	return adapter.baseAdapter.unsubscribeAll()
}
