package dataprovider_test

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"gitpct.epam.com/epmd-aepr/aos_vis/config"
	"gitpct.epam.com/epmd-aepr/aos_vis/dataprovider"
)

/*******************************************************************************
 * Init
 ******************************************************************************/

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		TimestampFormat:  "2006-01-02 15:04:05.000",
		FullTimestamp:    true})
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
}

/*******************************************************************************
 * Tests
 ******************************************************************************/

func TestPublicPath(t *testing.T) {
	provider, err := dataprovider.New(&config.Config{})
	if err != nil {
		t.Fatalf("Can't create data provider: %s", err)
	}

	path := "Attribute.Vehicle.VehicleIdentification.VIN"
	result, err := provider.IsPathPublic(path)
	if err != nil {
		t.Fatalf("Can't check path publicity: %s", err)
	}
	if !result {
		t.Errorf("Path %s should be public", path)
	}

	path = "Sensor.Engine.RPM"
	result, err = provider.IsPathPublic(path)
	if err != nil {
		t.Fatalf("Can't check path publicity: %s", err)
	}
	if result {
		t.Errorf("Path %s should not be public", path)
	}
}
