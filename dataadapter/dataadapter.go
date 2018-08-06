package dataadapter

type VisData struct {
	Path string
	Data interface{}
}

// DataAdapter interface for working with real data
type DataAdapter interface {
	StartGettingData(dataChan chan<- []VisData)
	SetData([]VisData) error
	Stop()
}

// GetVisDataAdapter create necessary adapter
func GetVisDataAdapter() DataAdapter {
	adapter, _ := NewTestAdapter()

	return adapter
}
