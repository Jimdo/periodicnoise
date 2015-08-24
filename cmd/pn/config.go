package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/vaughan0/go-ini"
)

// Locations of site/global config and per user config.
var (
	GlobalConfig = "/etc/periodicnoise/config.ini"
	UserConfig   = ".config/periodicnoise/config.ini"
)

// Load config from global and user-specific .ini file(s)
func loadConfig(name string) (ini.File, error) {
	c, err := ini.LoadFile(name)
	if err == nil {
		return c, nil
	}

	// if it doesn't exist, we don't just run with defaults
	if os.IsNotExist(err) {
		return make(ini.File), nil
	}
	return nil, err
}

func fillMonitoringCommands(config ini.File) {
	for result := range monitoringResults {
		if cmd, ok := config.Get("monitoring", result.String()); ok {
			monitoringCalls[result] = cmd
		}
	}
}

// Load monitoring commands from config
func loadMonitoringCommands() {
	global, err := loadConfig(GlobalConfig)
	if err != nil {
		log.Fatalln("ERROR: reading global config: ", err)
		return
	}
	fillMonitoringCommands(global)

	home := os.Getenv("HOME")

	user, err := loadConfig(filepath.Join(home, UserConfig))
	if err != nil {
		log.Fatalf("ERROR: reading per user config: %#v", err)
		return
	}
	fillMonitoringCommands(user)
}
