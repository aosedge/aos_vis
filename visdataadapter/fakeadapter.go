package visdataadapter

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

//FakeAdapter adapter for read data from sensorsender
type FakeAdapter struct {
	url string
}

/*******************************************************************************
 * Public
 ******************************************************************************/

//NewFakeAdapter Create wFakeAdapter
func NewFakeAdapter() (sensorAdapter *FakeAdapter) {
	sensorAdapter = new(FakeAdapter)
	return sensorAdapter
}

// StartGettingData start getting data with interval
func (sensorAdapter *FakeAdapter) StartGettingData(dataChan chan<- []VisData) {
	ticker := time.NewTicker(time.Duration(5) * time.Second)
	interrupt := make(chan os.Signal, 1) //TODO redo
	var RPM int
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			dataTosend := []VisData{}
			RPM += 50
			oneElement := VisData{Path: "Signal.Drivetrain.InternalCombustionEngine.RPM", Data: RPM}
			dataTosend = append(dataTosend, oneElement)
			oneElement = VisData{Path: "Attribute.Vehicle.UserIdentification.Users", Data: []string{"User2"}}
			dataTosend = append(dataTosend, oneElement)
			dataChan <- dataTosend
		case <-interrupt:
			log.Info("interrupt")
			break
		}
	}
}

// Stop stop getting data
func (sensorAdapter *FakeAdapter) Stop() {

}
