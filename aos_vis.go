package main

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"gitpct.epam.com/epmd-aepr/aos_vis/wsserver"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		TimestampFormat:  "2006-01-02 15:04:05.000",
		FullTimestamp:    true})
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
}

func main() {
	log.Info("Start Vehicle Information Service")
	server, err := wsserver.New("localhost:8088")
	if err != nil {
		log.Error("Can't create ws server ", err)
		return
	}

	go server.Start()

	// handle SIGTERM
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	server.Stop()
	os.Exit(1)
}
