package main

import (
	"code.google.com/p/goconf/conf"
	"log"
	"os"
	"path/filepath"
)

// Locations of site/global config and per user config.
var (
	GlobalConfig = "/etc/periodicnoise/config.ini"
	UserConfig   = ".config/periodicnoise/config.ini"
)

// Load config from global and user-specific .ini file(s)
func loadConfig() *conf.ConfigFile {
	c := conf.NewConfigFile()
	global, err := os.Open(GlobalConfig)
	if err == nil {
		defer global.Close()
		if err = c.Read(global); err != nil {
			log.Fatalln("ERROR: reading global config: ", err)
			return c
		}
	}

	home := os.Getenv("HOME")
	user, err := os.Open(filepath.Join(home, UserConfig))
	if err == nil {
		defer user.Close()
		if err = c.Read(user); err != nil {
			log.Fatalln("ERROR: reading per user config: ", err)
			return c
		}
	}

	return c
}

// Load monitoring commands from config
func loadMonitoringCommands() {
	config := loadConfig()
	states := []monitoringResult{monitorOk, monitorCritical, monitorWarning, monitorDebug, monitorUnknown}

	for _, state := range states {
		if cmd, err := config.GetString("monitoring", state.String()); err != nil {
			if _, ok := err.(conf.GetError); !ok {
				log.Fatalln("ERROR: reading monitoring commands: ", err)
			}
		} else {
			monitoringCalls[state] = cmd
		}
	}
}
