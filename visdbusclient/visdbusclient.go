package visdbusclient

import (
	"encoding/json"
	"github.com/godbus/dbus"
	log "github.com/sirupsen/logrus"
)

const (
	OBJECT_PATH    = "/com/aosservicemanager/vistoken"
	INTERFACE_NAME = "com.aosservicemanager.vistoken"
)

func GetVisPermissionByToken(token string) (permissions map[string]string, err error) {

	log.Info("GetVisPermissionByToken token ", token)

	var permissionJson string
	var dbusErr string

	conn, err := dbus.SessionBus()
	if err != nil {
		log.Error("No system bus conn ", err)
	}

	obj := conn.Object(INTERFACE_NAME, OBJECT_PATH)

	err = obj.Call(INTERFACE_NAME+".GetPermission", 0, token).Store(&permissionJson, &dbusErr)
	if err != nil {
		log.Error("can't make call ", err)
		return permissions, err
	}

	err = json.Unmarshal([]byte(permissionJson), &permissions)
	if err != nil {
		log.Error("Error Unmarshal  ", err)

		return permissions, err
	}

	log.Info(permissionJson)
	return permissions, nil
}
