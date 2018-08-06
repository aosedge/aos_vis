package main

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"gitpct.epam.com/epmd-aepr/aos_vis/config"
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
	log.Info("VIS Server")

	config, err := config.New("visconfig.json")
	if err != nil {
		log.Fatalf("Can' open config file: %s", err)
	}

	server, err := wsserver.New(config)
	if err != nil {
		log.Fatalf("Can't create ws server: %s", err)
	}

	// handle SIGTERM
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	server.Close()

	os.Exit(1)
}
