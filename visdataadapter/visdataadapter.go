package visdataadapter

//VisData TODO:
type VisData struct {
	Path string
	Data interface{}
}

//VisDataAdapter interface for working with real data
type VisDataAdapter interface {
	StartGettingData(dataChan chan<- []VisData)
	Stop()
}

//GetVisDataAdapter create necessary adapter
func GetVisDataAdapter() VisDataAdapter {
	adapter := NewSensorEmulatorAdapter("http://sensors:8800/stats")
	return adapter
}
