package dbusclient

import (
	"encoding/json"

	"github.com/godbus/dbus"
	log "github.com/sirupsen/logrus"
)

const (
	objectPath    = "/com/aosservicemanager/vistoken"
	interfaceName = "com.aosservicemanager.vistoken"
)

// GetVisPermissionByToken dbus call GetPermission
func GetVisPermissionByToken(token string) (permissions map[string]string, err error) {
	var permissionJSON string
	var dbusErr string

	conn, err := dbus.SessionBus()
	if err != nil {
		return permissions, err
	}

	obj := conn.Object(interfaceName, objectPath)

	if err = obj.Call(interfaceName+".GetPermission", 0, token).Store(&permissionJSON, &dbusErr); err != nil {
		return permissions, err
	}

	if err = json.Unmarshal([]byte(permissionJSON), &permissions); err != nil {
		return permissions, err
	}

	log.WithFields(log.Fields{"token": token, "permissions": permissions}).Debug("Get permissions")

	return permissions, nil
}
