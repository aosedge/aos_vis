package wsserver

import (
	"container/list"
	"context"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// WsServer websocket server structure
type WsServer struct {
	httpServer *http.Server
	upgrader   websocket.Upgrader
	mutex      sync.Mutex
	clients    *list.List
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// New creates new Web socket server
func New(addr, crt, key string) (server *WsServer, err error) {
	log.WithField("address", addr).Debug("Create wsserver")

	//TODO: add addr validation
	var localServer WsServer

	localServer.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     customCheckOrigin,
	}
	localServer.httpServer = &http.Server{Addr: addr}
	localServer.clients = list.New()

	http.HandleFunc("/", localServer.handleConnection)

	go func(crt, key string) {
		if err := localServer.httpServer.ListenAndServeTLS(crt, key); err != http.ErrServerClosed {
			log.Error("Server listening error: ", err)
			return
		}
	}(crt, key)

	server = &localServer

	return server, nil
}

// Close closes web socket server and all connections
func (server *WsServer) Close() {
	log.Debug("Stop server")

	server.mutex.Lock()
	defer server.mutex.Unlock()

	for element := server.clients.Front(); element != nil; element = element.Next() {
		element.Value.(*WsClientConnection).Close()
	}

	server.clients.Init()

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

	client, err := NewClientConn(wsConnection)
	if err != nil {
		log.Error("Can't create websocket client connection: ", err)
		wsConnection.Close()
		return
	}

	server.mutex.Lock()
	clientElement := server.clients.PushBack(client)
	server.mutex.Unlock()

	client.ProcessConnection()

	server.mutex.Lock()
	defer server.mutex.Unlock()
	for element := server.clients.Front(); element != nil; element = element.Next() {
		if element == clientElement {
			client.Close()
			server.clients.Remove(clientElement)
		}
	}
}
