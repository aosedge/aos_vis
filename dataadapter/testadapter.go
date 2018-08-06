package dataadapter

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// TestAdapter test adapter
type TestAdapter struct {
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// NewTestAdapter creates adapter to be used for tests
func NewTestAdapter() (adapter *TestAdapter, err error) {

	adapter = new(TestAdapter)

	return adapter, nil
}

// StartGettingData start getting data with interval
func (adapter *TestAdapter) StartGettingData(dataChan chan<- []VisData) {
	ticker := time.NewTicker(time.Duration(3) * time.Second)
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

// SetData sets data
func (adapter *TestAdapter) SetData([]VisData) error {
	return nil
}

// Stop stop getting data
func (adapter *TestAdapter) Stop() {

}
