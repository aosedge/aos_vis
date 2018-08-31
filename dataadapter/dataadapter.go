package dataadapter

import (
	"errors"
	"plugin"
)

// DataAdapter interface for working with real data
type DataAdapter interface {
	// GetName returns adapter name
	GetName() (name string)
	// GetPathList returns list of all pathes for this adapter
	GetPathList() (pathList []string, err error)
	// IsPathPublic returns true if requested data accessible without authorization
	IsPathPublic(path string) (result bool, err error)
	// GetData returns data by path
	GetData(pathList []string) (data map[string]interface{}, err error)
	// SetData sets data by pathes
	SetData(data map[string]interface{}) (err error)
	// GetSubscribeChannel returns channel on which data changes will be sent
	GetSubscribeChannel() (channel <-chan map[string]interface{})
	// Subscribe subscribes for data changes
	Subscribe(pathList []string) (err error)
	// Unsubscribe unsubscribes from data changes
	Unsubscribe(pathList []string) (err error)
	// UnsubscribeAll unsubscribes from all data changes
	UnsubscribeAll() (err error)
}

// NewAdapter creates new adapter instance
func NewAdapter(pluginPath string, configJSON []byte) (adapter DataAdapter, err error) {
	plugin, err := plugin.Open(pluginPath)
	if err != nil {
		return adapter, err
	}

	newAdapterSymbol, err := plugin.Lookup("NewAdapter")
	if err != nil {
		return adapter, err
	}

	newAdapterFunction, ok := newAdapterSymbol.(func(configJSON []byte) (DataAdapter, error))
	if !ok {
		return adapter, errors.New("Unexpected function type")
	}

	return newAdapterFunction(configJSON)
}
