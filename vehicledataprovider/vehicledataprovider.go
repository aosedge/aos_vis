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
	return "", errors.New("404 Not found")
}
