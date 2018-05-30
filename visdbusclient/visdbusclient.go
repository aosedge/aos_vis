package visdbusclient

import (
	"encoding/json"
	"github.com/godbus/dbus"
	log "github.com/sirupsen/logrus"
)

const (
	objectPath    = "/com/aosservicemanager/vistoken"
	interfaceName = "com.aosservicemanager.vistoken"
)

//GetVisPermissionByToken dbus call GetPermission
func GetVisPermissionByToken(token string) (permissions map[string]string, err error) {
	log.Info("GetVisPermissionByToken token ", token)

	var permissionJSON string
	var dbusErr string

	conn, err := dbus.SessionBus()
	if err != nil {
		log.Error("No system bus conn ", err)
		return permissions, err
	}

	obj := conn.Object(interfaceName, objectPath)

	err = obj.Call(interfaceName+".GetPermission", 0, token).Store(&permissionJSON, &dbusErr)
	if err != nil {
		log.Error("Can't make call ", err)
		return permissions, err
	}

	err = json.Unmarshal([]byte(permissionJSON), &permissions)
	if err != nil {
		log.Error("Error Unmarshal  ", err)

		return permissions, err
	}

	log.Info(permissionJSON)
	return permissions, nil
}
