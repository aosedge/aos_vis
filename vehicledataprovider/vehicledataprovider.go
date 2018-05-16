package vehicledataprovider

import (
	"errors"
)

func IsPublicPath(path string) bool {
	if path == "Attribute.Vehicle.VehicleIdentification.VIN" {
		return true
	}
	return false
}

func GetDataByPath(path string) (interface{}, error) {
	if path == "Attribute.Vehicle.VehicleIdentification.VIN" {
		return "1234567890QWERTYU", nil
	}
	if path == "Signal.Drivetrain.InternalCombustionEngine.RPM" {
		return 2372, nil
	}
	return "", errors.New("404 Not found")
}

func RegestrateSubscriptionClient(subasChan chan interface{}, path string) (string, error) {
	return "1111", nil
}

func RegestrateUnSubscription(subasChan chan interface{}, subscID string) (err error) {
	err = nil
	if subscID != "1111" {
		err = errors.New("404 Not found")
	}
	return err
}

func RegestrateUnSubscribAll(subasChan chan interface{}) (err error) {
	return nil
}
