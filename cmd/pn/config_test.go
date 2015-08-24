package main

import (
	"fmt"
	"testing"
)

func init() {
	// Use config test fixture
	GlobalConfig = "testdata/config.ini"
	// Make sure user config does not overwrite test data
	UserConfig = ".ini"
}

func makeMonitoringCommand(result monitoringResult) string {
	return fmt.Sprintf("send_nsca \"%%(event): [%s] %%(message)\"", result)
}

func TestHasMonitoringCommands(t *testing.T) {
	config, err := loadConfig(GlobalConfig)
	if err != nil {
		t.Fatal(err)
	}
	options := config.Section("monitoring")

	if len(options) != len(monitoringResults) {
		t.Errorf("got %d, want %d", len(options), len(monitoringResults))
		return
	}

	for result := range monitoringResults {
		cmd, _ := config.Get("monitoring", result.String())
		want := makeMonitoringCommand(result)
		if cmd != want {
			t.Errorf("got %s, want %s", cmd, want)
			return
		}
	}
}

func TestLoadsMonitoringCommands(t *testing.T) {
	loadMonitoringCommands()

	for result := range monitoringResults {
		cmd := monitoringCalls[result]
		want := makeMonitoringCommand(result)
		if cmd != want {
			t.Errorf("got %s, want %s", cmd, want)
			return
		}
	}
}
