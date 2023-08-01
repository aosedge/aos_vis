// SPDX-License-Identifier: Apache-2.0
//
// Copyright (C) 2021 Renesas Electronics Corporation.
// Copyright (C) 2021 EPAM Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/aoscloud/aos_common/aoserrors"
	"github.com/coreos/go-systemd/daemon"
	"github.com/coreos/go-systemd/journal"
	log "github.com/sirupsen/logrus"

	"github.com/aoscloud/aos_vis/config"
	"github.com/aoscloud/aos_vis/permissionprovider"
	_ "github.com/aoscloud/aos_vis/plugins"
	"github.com/aoscloud/aos_vis/visserver"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

type journalHook struct {
	severityMap map[log.Level]journal.Priority
}

/*******************************************************************************
 * Vars
 ******************************************************************************/

// GitSummary provided by govvv at compile-time.
var GitSummary string //nolint:gochecknoglobals

/*******************************************************************************
 * Init
 ******************************************************************************/

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		TimestampFormat:  "2006-01-02 15:04:05.000",
		FullTimestamp:    true,
	})
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stdout)
}

/*******************************************************************************
 * Journal hook
 ******************************************************************************/

func newJournalHook() (hook *journalHook) {
	hook = &journalHook{
		severityMap: map[log.Level]journal.Priority{
			log.DebugLevel: journal.PriDebug,
			log.InfoLevel:  journal.PriInfo,
			log.WarnLevel:  journal.PriWarning,
			log.ErrorLevel: journal.PriErr,
			log.FatalLevel: journal.PriCrit,
			log.PanicLevel: journal.PriEmerg,
		},
	}

	return hook
}

func (hook *journalHook) Fire(entry *log.Entry) (err error) {
	if entry == nil {
		return aoserrors.New("log entry is nil")
	}

	logMessage, err := entry.String()
	if err != nil {
		return aoserrors.Wrap(err)
	}

	err = journal.Print(hook.severityMap[entry.Level], logMessage)

	return aoserrors.Wrap(err)
}

func (hook *journalHook) Levels() []log.Level {
	return []log.Level{
		log.PanicLevel,
		log.FatalLevel,
		log.ErrorLevel,
		log.WarnLevel,
		log.InfoLevel,
		log.DebugLevel,
	}
}

/*******************************************************************************
 * Main
 ******************************************************************************/

func main() {
	// Initialize command line flags
	configFile := flag.String("c", "visconfig.json", "path to config file")
	strLogLevel := flag.String("v", "info", `log level: "debug", "info", "warn", "error", "fatal", "panic"`)
	showVersion := flag.Bool("version", false, `show VIS version`)
	useJournal := flag.Bool("j", false, "output logs to systemd journal")

	flag.Parse()

	// Show versions
	if *showVersion {
		fmt.Printf("Version: %s\n", GitSummary) //nolint:forbidigo
		return
	}

	// Set log output
	if *useJournal {
		log.AddHook(newJournalHook())
		log.SetOutput(io.Discard)
	} else {
		log.SetOutput(os.Stdout)
	}

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

	permissionsProvider, err := permissionprovider.New(config, false)
	if err != nil {
		log.Fatalf("Can't create permission provider: %s", err)
	}

	server, err := visserver.New(config, permissionsProvider)
	if err != nil {
		log.Fatalf("Can't create ws server: %s", err)
	}

	// Notify systemd
	if _, err = daemon.SdNotify(false, daemon.SdNotifyReady); err != nil {
		log.Errorf("Can't notify systemd: %s", err)
	}

	// handle SIGTERM
	c := make(chan os.Signal, 2) //nolint:gomnd
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	server.Close()

	permissionsProvider.Close()
}
