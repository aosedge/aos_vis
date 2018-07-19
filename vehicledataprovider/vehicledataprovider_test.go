package vehicledataprovider_test

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"gitpct.epam.com/epmd-aepr/aos_vis/vehicledataprovider"
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

	dataprovider := vehicledataprovider.GetInstance()
	dataChannel := make(chan vehicledataprovider.SubscriptionOutputData)
	idStr, err := dataprovider.RegestrateSubscriptionClient(dataChannel, "*")
	if err != nil {
		t.Error("error subscription")
	}

	idStr, err = dataprovider.RegestrateSubscriptionClient(dataChannel, "Signal.*")
	if err != nil {
		t.Error("error subscription")
	}
	log.Debug("[TEST] Subscription to Signal* OK id=", idStr)

	incomedata := <-dataChannel
	log.Debug("[TEST] receive data from channel ", incomedata)

}
