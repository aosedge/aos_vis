package wsserver

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// WsServer websocket server structure
type WsServer struct {
	addr       string
	httpServer *http.Server
	upgrader   websocket.Upgrader
	crt        string
	key        string
	//TODO: add list with client connections
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// New creates new Web socket server
func New(addr, crt, key string) (server *WsServer, err error) {
	log.WithField("address", addr).Debug("Create wsserver")

	//TODO: add addr validation
	var localServer WsServer
	localServer.addr = addr
	localServer.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     customCheckOrigin,
	}

	localServer.crt = crt
	localServer.key = key
	localServer.httpServer = &http.Server{Addr: addr}

	server = &localServer

	return server, nil
}

// Start start web socket server
func (server *WsServer) Start() {
	log.Info("Start server")
	http.HandleFunc("/", server.handleConnection)

	if err := server.httpServer.ListenAndServeTLS(server.crt, server.key); err != http.ErrServerClosed {
		log.Error("Server listening error: ", err)
	}
}

// Stop web socket server
func (server *WsServer) Close() {
	log.Debug("Stop server")

	//TODO: close all connections
	server.httpServer.Shutdown(context.Background())
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func customCheckOrigin(r *http.Request) bool {
	return true
}

func (server *WsServer) handleConnection(w http.ResponseWriter, r *http.Request) {
	log.WithField("RemoteAddr", r.RemoteAddr).Debug("New connection request")

	if websocket.IsWebSocketUpgrade(r) != true {
		log.Error("New connection is not websocket")
		return
	}

	wsConnection, err := server.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("Can't make websocket connection: ", err)
		return
	}
	defer wsConnection.Close()

	client, err := NewClientConn(wsConnection)
	if err != nil {
		log.Error("Can't create websocket client connection: ", err)
		return
	}

	client.ProcessConnection()
}
