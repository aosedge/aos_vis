package dbusclient_test

import (
	"os"
	"testing"

	"github.com/godbus/dbus"
	log "github.com/sirupsen/logrus"
	"gitpct.epam.com/epmd-aepr/aos_vis/dbusclient"
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
	return `{"*": "rw", "123": "rw"}`, "OK", nil
}

/*******************************************************************************
 * Main
 ******************************************************************************/

func TestMain(m *testing.M) {
	conn, err := dbus.SessionBus()
	if err != nil {
		log.Fatalf("Can't create session connection: %v", err)
	}

	reply, err := conn.RequestName("com.aosservicemanager.vistoken", dbus.NameFlagDoNotQueue)
	if err != nil {
		log.Fatal("Can't request name")
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		log.Fatal("Name already taken")
	}

	server := dbusInterface{}
	conn.Export(server, "/com/aosservicemanager/vistoken", "com.aosservicemanager.vistoken")

	ret := m.Run()

	os.Exit(ret)
}

func TestDBUS(t *testing.T) {
	permission, err := dbusclient.GetVisPermissionByToken("APPID")
	if err != nil {
		t.Fatalf("Can't make D-Bus call: %s", err)
	}

	if len(permission) != 2 {
		t.Fatal("Permission list length !=2")
	}

	if permission["*"] != "rw" {
		t.Fatal("Incorrect permissions")
	}
}
