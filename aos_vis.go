package main

import (
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"gitpct.epam.com/epmd-aepr/aos_vis/wsserver"
)

type configuration struct {
	ServerUrl string
	VISCert   string
	VISKey    string
}

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		TimestampFormat:  "2006-01-02 15:04:05.000",
		FullTimestamp:    true})
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
}

func main() {
	log.Info("main")
	file, err := os.Open("visconfig.json")
	if err != nil {
		log.Fatal("Error while opening fcrypt configurataion file: ", err)
	}
	var config configuration

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		log.Error("Erro while parsing visconfig.json: ", err)
		return
	}

	log.Info("ServerURl:  ", config.ServerUrl)
	log.Info("VISCert:    ", config.VISCert)
	log.Info("VISKey:     ", config.VISKey)

	server, err := wsserver.New(config.ServerUrl, config.VISCert, config.VISKey)
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
