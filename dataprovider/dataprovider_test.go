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

func TestDBUS(t *testing.T) {
	log.Debug("[TEST] TestGet")

	provider, err := dataprovider.New(&config.Config{})

	dataChannel := make(chan dataprovider.SubscriptionOutputData)
	idStr, err := provider.Subscribe(dataChannel, "*")
	if err != nil {
		t.Error("error subscription")
	}

	idStr, err = provider.Subscribe(dataChannel, "Signal.*")
	if err != nil {
		t.Error("error subscription")
	}
	log.Debug("[TEST] Subscription to Signal* OK id=", idStr)

	incomedata := <-dataChannel
	log.Debug("[TEST] receive data from channel ", incomedata)

}
