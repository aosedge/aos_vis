package dataadapter

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
