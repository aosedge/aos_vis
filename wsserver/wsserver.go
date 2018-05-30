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

//WsServer websocket server structure
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

//New creates new Web socket server
func New(addr, crt, key string) (server *WsServer, err error) {
	log.Debug("wsserver creation ", addr)
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

//Start start web socket server
func (server *WsServer) Start() {
	log.Info("Start server")
	http.HandleFunc("/", server.handleConnection)

	if err := server.httpServer.ListenAndServeTLS(server.crt, server.key); err != http.ErrServerClosed {
		log.Debug("Httpserver: ListenAndServe() error: ", err)
	}
}

//Stop web socket server
func (server *WsServer) Stop() {
	log.Info("Stop server!!")
	//TODO: close all connections
	server.httpServer.Shutdown(context.Background())
	//server.httpServer.Close()
}

/*******************************************************************************
 * Private
 ******************************************************************************/
func customCheckOrigin(r *http.Request) bool {
	return true
}

func (server *WsServer) handleConnection(w http.ResponseWriter, r *http.Request) {
	log.Debug("New connection ")
	if websocket.IsWebSocketUpgrade(r) != true {
		log.Warning("New connection is not websocket")
		return
	}

	c, err := server.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("Can't make websocket coinnection :", err)
		return
	}

	defer c.Close()

	wsClientCon, err := NewClientConn(c)
	if err != nil {
		log.Error("Can't create ws client connection :", err)
		return
	}
	wsClientCon.ProcessConnection()
	log.Debug("Stop handleConnection")
}
