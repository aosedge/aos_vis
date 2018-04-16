package wsserver

import (
	"errors"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type WsClientConnection struct {
	name         string
	wsConn       *websocket.Conn
	isAuthorised bool
}

func NewClientConn(wsConn *websocket.Conn) (wsClientCon *WsClientConnection, err error) {
	log.Debug("NewClientConn")
	if wsConn == nil {
		return wsClientCon, errors.New("Null connection")
	}
	var localConnection WsClientConnection
	localConnection.wsConn = wsConn
	wsClientCon = &localConnection

	return wsClientCon, nil
}

//ProcessConnection process incommming websoket connection
func (client *WsClientConnection) ProcessConnection() {
	//TODO: add select fro processing subscription channel
	log.Debug("Start processing new WS connection")
	for {
		mt, message, err := client.wsConn.ReadMessage()
		if err != nil {
			log.Warning("Can't read from WS: ", err)
			break
		}

		log.Debug("message type: ", mt)
		log.Info("recv: ", string(message))
		client.WriteMessage(message)

	}
	log.Debug("Stop processConnection")
}

func (client *WsClientConnection) WriteMessage(data []byte) {
	err := client.wsConn.WriteMessage(1, data)
	if err != nil {
		log.Error("Can't write to WS: ", err)
	}
}
