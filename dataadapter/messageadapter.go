package dataadapter

import (
	log "github.com/sirupsen/logrus"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// MessageAdapter message adapter
type MessageAdapter struct {
	baseAdapter *BaseAdapter
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// NewMessageAdapter creates adapter to display messages
func NewMessageAdapter() (adapter *MessageAdapter, err error) {
	log.Info("Create message adapter")

	adapter = new(MessageAdapter)

	adapter.baseAdapter, err = newBaseAdapter()
	if err != nil {
		return nil, err
	}

	adapter.baseAdapter.data["Attribute.Car.Message"] = &baseData{}

	return adapter, nil
}

// GetName returns adapter name
func (adapter *MessageAdapter) GetName() (name string) {
	return "MessageAdapter"
}

// GetPathList returns list of all pathes for this adapter
func (adapter *MessageAdapter) GetPathList() (pathList []string, err error) {
	return adapter.baseAdapter.getPathList()
}

// IsPathPublic returns true if requested data accessible without authorization
func (adapter *MessageAdapter) IsPathPublic(path string) (result bool, err error) {
	adapter.baseAdapter.mutex.Lock()
	defer adapter.baseAdapter.mutex.Unlock()

	// TODO: change to false once authorization is intergrated

	return true, nil
}

// GetData returns data by path
func (adapter *MessageAdapter) GetData(pathList []string) (data map[string]interface{}, err error) {
	return adapter.baseAdapter.getData(pathList)
}

// SetData sets data by pathes
func (adapter *MessageAdapter) SetData(data map[string]interface{}) (err error) {
	if err = adapter.baseAdapter.setData(data); err != nil {
		return err
	}

	for path, value := range data {
		log.Infof("%s = %v", path, value)
	}

	return nil
}

// GetSubscribeChannel returns channel on which data changes will be sent
func (adapter *MessageAdapter) GetSubscribeChannel() (channel <-chan map[string]interface{}) {
	return adapter.baseAdapter.subscribeChannel
}

// Subscribe subscribes for data changes
func (adapter *MessageAdapter) Subscribe(pathList []string) (err error) {
	return adapter.baseAdapter.subscribe(pathList)
}

// Unsubscribe unsubscribes from data changes
func (adapter *MessageAdapter) Unsubscribe(pathList []string) (err error) {
	return adapter.baseAdapter.unsubscribe(pathList)
}

// UnsubscribeAll unsubscribes from all data changes
func (adapter *MessageAdapter) UnsubscribeAll() (err error) {
	return adapter.baseAdapter.unsubscribeAll()
}
