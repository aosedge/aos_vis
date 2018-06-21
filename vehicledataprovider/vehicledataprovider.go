package vehicledataprovider

import (
	"errors"
	"sync"
)

//
type VisData struct {
	path string
	data interface{}
}

// VehicleDataProvider interface for geeting vehicle data
type VehicleDataProvider struct {
	sensorDataChannel <-chan VisData
}

var instance *VehicleDataProvider
var once sync.Once

// GetInstance  get pointer to VehicleDataProvider
func GetInstance() *VehicleDataProvider {
	once.Do(func() {
		instance = &VehicleDataProvider{}
		instance.sensorDataChannel = make(chan VisData, 100)
		instance.start()
	})
	return instance
}

func (dataprovider *VehicleDataProvider) start() {

}

// IsPublicPath if path is public no authentication is required
func (dataprovider *VehicleDataProvider) IsPublicPath(path string) bool {
	if path == "Attribute.Vehicle.VehicleIdentification.VIN" {
		return true
	}
	if path == "Attribute.Vehicle.UserIdentification.Users" {
		return true
	}
	return false
}

// GetDataByPath get vehicle data by path
func (dataprovider *VehicleDataProvider) GetDataByPath(path string) (interface{}, error) {
	if path == "Attribute.Vehicle.VehicleIdentification.VIN" {
		return "1234567890QWERTYU", nil
	}
	if path == "Signal.Drivetrain.InternalCombustionEngine.RPM" {
		return 2372, nil
	}
	if path == "Attribute.Vehicle.UserIdentification.Users" {
		return []string{"User1"}, nil
	}
	return "", errors.New("404 Not found")
}

// RegestrateSubscriptionClient TODO
func (dataprovider *VehicleDataProvider) RegestrateSubscriptionClient(subsChan chan interface{}, path string) (string, error) {
	return "1111", nil
}

// RegestrateUnSubscription TODO
func (dataprovider *VehicleDataProvider) RegestrateUnSubscription(subsChan chan interface{}, subsID string) (err error) {
	err = nil
	if subsID != "1111" {
		err = errors.New("404 Not found")
	}
	return err
}

// RegestrateUnSubscribAll TODO
func (dataprovider *VehicleDataProvider) RegestrateUnSubscribAll(subsChan chan interface{}) (err error) {
	return nil
}
