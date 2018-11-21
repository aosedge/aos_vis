package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"gitpct.epam.com/epmd-aepr/aos_vis/config"
	"gitpct.epam.com/epmd-aepr/aos_vis/wsserver"
)

// GitSummary provided by govvv at compile-time
var GitSummary string

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		TimestampFormat:  "2006-01-02 15:04:05.000",
		FullTimestamp:    true})
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stdout)
}

func main() {
	// Initialize command line flags
	configFile := flag.String("c", "visconfig.json", "path to config file")
	strLogLevel := flag.String("v", "info", `log level: "debug", "info", "warn", "error", "fatal", "panic"`)

	flag.Parse()

	// Set log level
	logLevel, err := log.ParseLevel(*strLogLevel)
	if err != nil {
		log.Fatalf("Error: %s", err)
	}
	log.SetLevel(logLevel)

	log.WithFields(log.Fields{"configFile": *configFile, "version": GitSummary}).Info("Start VIS Server")

	// Create config
	config, err := config.New(*configFile)
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
