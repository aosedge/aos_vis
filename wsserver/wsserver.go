package wsserver

import (
	"container/list"
	"context"
	"net/http"
	"sync"

	"gitpct.epam.com/epmd-aepr/aos_vis/dataprovider"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"gitpct.epam.com/epmd-aepr/aos_vis/config"
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

	dataProvider *dataprovider.DataProvider
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// New creates new Web socket server
func New(config *config.Config) (server *WsServer, err error) {
	log.Debug("Create wsserver")

	//TODO: add addr validation
	var localServer WsServer

	localServer.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	if localServer.dataProvider, err = dataprovider.New(config); err != nil {
		return server, err
	}

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", localServer.handleConnection)

	localServer.httpServer = &http.Server{Addr: config.ServerURL, Handler: serveMux}
	localServer.clients = list.New()

	go func(crt, key string) {
		log.WithFields(log.Fields{"address": config.ServerURL, "crt": crt, "key": key}).Debug("Listen for VIS clients")

		if err := localServer.httpServer.ListenAndServeTLS(crt, key); err != http.ErrServerClosed {
			log.Error("Server listening error: ", err)
			return
		}
	}(config.VISCert, config.VISKey)

	server = &localServer

	return server, nil
}

// Close closes web socket server and all connections
func (server *WsServer) Close() {
	log.Debug("Stop server")

	server.mutex.Lock()
	defer server.mutex.Unlock()

	for element := server.clients.Front(); element != nil; element = element.Next() {
		element.Value.(*wsClient).close()
	}

	server.httpServer.Shutdown(context.Background())

	server.dataProvider.Close()
}

/*******************************************************************************
 * Private
 ******************************************************************************/

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

	client, err := newClient(wsConnection, server.dataProvider)
	if err != nil {
		log.Error("Can't create websocket client connection: ", err)
		wsConnection.Close()
		return
	}

	server.mutex.Lock()
	clientElement := server.clients.PushBack(client)
	server.mutex.Unlock()

	client.run()

	server.mutex.Lock()
	defer server.mutex.Unlock()
	for element := server.clients.Front(); element != nil; element = element.Next() {
		if element == clientElement {
			client.close()
			server.clients.Remove(clientElement)
		}
	}
}
