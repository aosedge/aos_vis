package dataadapter

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// SensorEmulatorAdapter adapter for read data from sensorsender
type SensorEmulatorAdapter struct {
	url string
}

const (
	updatePeriod = 500
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
			visData, err := convertDataToVisFormat(jsonData)
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

// SetData sets VIS data
func (sensorAdapter *SensorEmulatorAdapter) SetData(data []VisData) (err error) {
	sendData, err := convertVisFormatToData(data)
	if err != nil {
		return err
	}

	log.Debugf("Send data: %s", string(sendData))

	res, err := http.Post(sensorAdapter.url, "application/json", bytes.NewReader(sendData))
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return errors.New(res.Status)
	}

	return nil
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

func convertDataToVisFormat(jsonData []byte) (visData []VisData, err error) {
	var data interface{}

	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return visData, err
	}

	parseNode("Signal.Emulator", data, &visData)

	return visData, nil
}

func convertVisFormatToData(visData []VisData) (jsonData []byte, err error) {
	sendData := make(map[string]interface{})

	for _, item := range visData {
		if strings.HasPrefix(item.Path, "Attribute.Emulator.") {
			item.Path = strings.TrimPrefix(item.Path, "Attribute.Emulator.")
			sendData[item.Path] = item.Data
		} else {
			log.Warningf("Skip %s item", item.Path)
		}
	}

	jsonData, err = json.Marshal(&sendData)
	if err != nil {
		return jsonData, err
	}

	return jsonData, nil
}
