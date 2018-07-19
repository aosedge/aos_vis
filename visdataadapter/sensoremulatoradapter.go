package visdataadapter

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

// SensorEmulatorAdapter adapter for read data from sensorsender
type SensorEmulatorAdapter struct {
	url string
}

const (
	updatePeriod = 1000
)

/*******************************************************************************
 * Public
 ******************************************************************************/

// NewSensorEmulatorAdapter Create SensorEmulatorAdapter
func NewSensorEmulatorAdapter(url string) (sensorAdapter *SensorEmulatorAdapter) {
	sensorAdapter = new(SensorEmulatorAdapter)
	sensorAdapter.url = url
	return sensorAdapter
}

// StartGettingData start getting data with interval
func (sensorAdapter *SensorEmulatorAdapter) StartGettingData(dataChan chan<- []VisData) {
	ticker := time.NewTicker(time.Duration(updatePeriod) * time.Millisecond)
	interrupt := make(chan os.Signal, 1) //TODO redo
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			log.Info("Send GET request")
			jsonData, err := sensorAdapter.readDataFromSensors()
			if err != nil {
				log.Error("Can't read data: ", err)
				continue
			}
			visData, err := sensorAdapter.convertDataToVisFormat(jsonData)
			if err != nil {
				log.Error("Can't convert to vis data: ", err)
				continue
			}
			dataChan <- visData

		case <-interrupt:
			log.Info("interrupt")
			break
		}
	}
}

// Stop TODO
func (sensorAdapter *SensorEmulatorAdapter) Stop() {

}

/*******************************************************************************
 * Private
 ******************************************************************************/

func (sensorAdapter *SensorEmulatorAdapter) readDataFromSensors() (data []byte, err error) {
	res, err := http.Get(sensorAdapter.url)
	if err != nil {
		log.Error("Error HTTP GET to ", sensorAdapter.url, err)
		return data, err
	}
	data, err = ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Error("Error read from HTTP body responce ", err)
		return data, err
	}

	return data, nil
}

func parseNode(prefix string, element interface{}, visData *[]VisData) {
	m, ok := element.(map[string]interface{})
	if ok {
		for path, value := range m {
			parseNode(prefix+"."+path, value, visData)
		}
	} else {
		visElement := VisData{Path: prefix, Data: element}
		*visData = append(*visData, visElement)
	}
}

func (sensorAdapter *SensorEmulatorAdapter) convertDataToVisFormat(jsonData []byte) (visData []VisData, err error) {
	var data interface{}

	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return visData, err
	}

	parseNode("Signal.Emulator", data, &visData)

	return visData, nil
}
