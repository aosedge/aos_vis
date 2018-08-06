package dataadapter

// VisData represents single VIS node
type VisData struct {
	Path string
	Data interface{}
}

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
}
