package visdbusclient_test

import (
	"github.com/godbus/dbus"
	log "github.com/sirupsen/logrus"
	"os"
	"testing"

	dbusclient "gitpct.epam.com/epmd-aepr/aos_vis/visdbusclient"
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

type dbusInterface struct {
}

func (GetPermission dbusInterface) GetPermission(token string) (string, string, *dbus.Error) {
	log.Info("[TEST] GetPermission token: ", token)
	//return `[{"*":"rw"}]`, "OK", nil
	return `{"*": "rw", "123": "rw"}`, "OK", nil
}

/*******************************************************************************
 * Main
 ******************************************************************************/

func TestMain(m *testing.M) {

	conn, err := dbus.SessionBus()
	if err != nil {
		panic(err)
	}
	reply, err := conn.RequestName("com.aosservicemanager.vistoken",
		dbus.NameFlagDoNotQueue)
	if err != nil {
		log.Error("can't RequestName")
		os.Exit(1)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		log.Error("name already taken")
		os.Exit(1)
	}
	server := dbusInterface{}
	conn.Export(server, "/com/aosservicemanager/vistoken", "com.aosservicemanager.vistoken")

	ret := m.Run()

	os.Exit(ret)
}

func TestDBUS(t *testing.T) {
	log.Debug("[TEST] TestGet")

	permission, err := dbusclient.GetVisPermissionByToken("APPID")
	if err != nil {
		t.Fatalf("Can't make d-bus call %v", err)
		return
	}

	if len(permission) != 2 {
		t.Fatalf("Permission list length !=2 ")
		return
	}

	if permission["*"] != "rw" {
		t.Fatalf("Incorrect permisson")
		return
	}

}
