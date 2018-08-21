package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gitpct.epam.com/epmd-aepr/aos_vis/dataadapter"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

const (
	defaultUpdatePeriod = 500
)

// SensorEmulatorAdapter sensor emulator adapter
type SensorEmulatorAdapter struct {
	sensorURL    *url.URL
	updatePeriod uint64

	baseAdapter *dataadapter.BaseAdapter
}

type config struct {
	SensorURL    string
	UpdatePeriod uint64
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// NewAdapter creates adapter instance
func NewAdapter(configJSON []byte) (adapter dataadapter.DataAdapter, err error) {
	log.Info("Create sensor emulator adapter")

	localAdapter := new(SensorEmulatorAdapter)

	cfg := config{UpdatePeriod: defaultUpdatePeriod}

	// Parse config
	err = json.Unmarshal(configJSON, &cfg)
	if err != nil {
		return nil, err
	}

	if cfg.SensorURL == "" {
		return nil, errors.New("Sensor URL should be defined")
	}

	localAdapter.updatePeriod = cfg.UpdatePeriod
	localAdapter.sensorURL, err = url.Parse(cfg.SensorURL)

	if localAdapter.baseAdapter, err = dataadapter.NewBaseAdapter(); err != nil {
		return nil, err
	}

	localAdapter.baseAdapter.Name = "SensorEmulatorAdapter"

	// Create data map
	data, err := localAdapter.getDataFromSensorEmulator()
	if err != nil {
		return nil, err
	}
	for path, value := range data {
		localAdapter.baseAdapter.Data[path] = &dataadapter.BaseData{Value: value}
	}

	// Create attributes
	localAdapter.baseAdapter.Data["Attribute.Emulator.rectangle_long0"] = &dataadapter.BaseData{}
	localAdapter.baseAdapter.Data["Attribute.Emulator.rectangle_lat0"] = &dataadapter.BaseData{}
	localAdapter.baseAdapter.Data["Attribute.Emulator.rectangle_long1"] = &dataadapter.BaseData{}
	localAdapter.baseAdapter.Data["Attribute.Emulator.rectangle_lat1"] = &dataadapter.BaseData{}
	localAdapter.baseAdapter.Data["Attribute.Emulator.to_rectangle"] = &dataadapter.BaseData{}
	localAdapter.baseAdapter.Data["Attribute.Emulator.stop"] = &dataadapter.BaseData{}
	localAdapter.baseAdapter.Data["Attribute.Emulator.tire_break"] = &dataadapter.BaseData{}

	go localAdapter.processData()

	return localAdapter, nil
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// GetName returns adapter name
func (adapter *SensorEmulatorAdapter) GetName() (name string) {
	return adapter.baseAdapter.GetName()
}

// GetPathList returns list of all pathes for this adapter
func (adapter *SensorEmulatorAdapter) GetPathList() (pathList []string, err error) {
	return adapter.baseAdapter.GetPathList()
}

// IsPathPublic returns true if requested data accessible without authorization
func (adapter *SensorEmulatorAdapter) IsPathPublic(path string) (result bool, err error) {
	adapter.baseAdapter.Mutex.Lock()
	defer adapter.baseAdapter.Mutex.Unlock()

	// TODO: return false, once authorization is integrated

	return true, nil
}

// GetData returns data by path
func (adapter *SensorEmulatorAdapter) GetData(pathList []string) (data map[string]interface{}, err error) {
	return adapter.baseAdapter.GetData(pathList)
}

// SetData sets data by pathes
func (adapter *SensorEmulatorAdapter) SetData(data map[string]interface{}) (err error) {
	sendData, err := convertVisFormatToData(data)
	if err != nil {
		return err
	}

	path, err := url.Parse("attributes/")
	if err != nil {
		return err
	}

	address := adapter.sensorURL.ResolveReference(path).String()

	log.WithField("url", address).Debugf("Set data to sensor emulator: %s", string(sendData))

	res, err := http.Post(address, "application/json", bytes.NewReader(sendData))
	if err != nil {
		return err
	}
	if res.StatusCode != 201 {
		return errors.New(res.Status)
	}

	return adapter.baseAdapter.SetData(data)
}

// GetSubscribeChannel returns channel on which data changes will be sent
func (adapter *SensorEmulatorAdapter) GetSubscribeChannel() (channel <-chan map[string]interface{}) {
	return adapter.baseAdapter.SubscribeChannel
}

// Subscribe subscribes for data changes
func (adapter *SensorEmulatorAdapter) Subscribe(pathList []string) (err error) {
	return adapter.baseAdapter.Subscribe(pathList)
}

// Unsubscribe unsubscribes from data changes
func (adapter *SensorEmulatorAdapter) Unsubscribe(pathList []string) (err error) {
	return adapter.baseAdapter.Unsubscribe(pathList)
}

// UnsubscribeAll unsubscribes from all data changes
func (adapter *SensorEmulatorAdapter) UnsubscribeAll() (err error) {
	return adapter.baseAdapter.UnsubscribeAll()
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func parseNode(prefix string, element interface{}) (visData map[string]interface{}) {
	visData = make(map[string]interface{})

	m, ok := element.(map[string]interface{})
	if ok {
		for path, value := range m {
			for visPath, visValue := range parseNode(prefix+"."+path, value) {
				visData[visPath] = visValue
			}
		}
	} else {
		visData[prefix] = element
	}

	return visData
}

func convertDataToVisFormat(dataJSON []byte) (visData map[string]interface{}, err error) {
	var data interface{}

	err = json.Unmarshal(dataJSON, &data)
	if err != nil {
		return visData, err
	}

	visData = parseNode("Signal.Emulator", data)

	return visData, nil
}

func (adapter *SensorEmulatorAdapter) getDataFromSensorEmulator() (visData map[string]interface{}, err error) {
	path, err := url.Parse("stats")
	if err != nil {
		return visData, err
	}

	address := adapter.sensorURL.ResolveReference(path).String()

	res, err := http.Get(address)
	if err != nil {
		return visData, err
	}

	data, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return visData, err
	}

	log.WithField("url", address).Debugf("Get data from sensor emulator: %s", string(data))

	return convertDataToVisFormat(data)
}

func (adapter *SensorEmulatorAdapter) processData() {
	ticker := time.NewTicker(time.Duration(adapter.updatePeriod) * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			data, err := adapter.getDataFromSensorEmulator()
			if err != nil {
				log.Errorf("Can't read data: %s", err)
				continue
			}
			if err = adapter.baseAdapter.SetData(data); err != nil {
				log.Errorf("Can't update data: %s", err)
				continue
			}
		}
	}
}

func convertVisFormatToData(visData map[string]interface{}) (dataJSON []byte, err error) {
	sendData := make(map[string]interface{})

	for path, value := range visData {
		if strings.HasPrefix(path, "Attribute.Emulator.") {
			path = strings.TrimPrefix(path, "Attribute.Emulator.")
			sendData[path] = value
		} else {
			return dataJSON, fmt.Errorf("Path %s does not exist", path)
		}
	}

	dataJSON, err = json.Marshal(&sendData)
	if err != nil {
		return dataJSON, err
	}

	return dataJSON, nil
}
