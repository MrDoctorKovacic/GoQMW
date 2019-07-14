package main

import (
	"encoding/json"
	"flag"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/bluetooth"
	"github.com/MrDoctorKovacic/MDroid-Core/influx"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/MrDoctorKovacic/MDroid-Core/status"
)

// Config controls program settings and general persistent settings
type Config struct {
	DatabaseHost     string
	DatabaseName     string
	BluetoothAddress string
	PingHost         string
	DebugSessionFile string
}

// Configure verbose output for code in package main
var VERBOSE_OUTPUT bool

// Timezone location for session last used and logging
var TIMEZONE *time.Location

// If we're using a database at all
var DATABASE_ENABLED bool

// Database variable, that may or may not be used globally
var DB influx.Influx

func parseConfig() string {
	// Start with program arguments
	var (
		settingsFile string
		sessionFile  string // This is for debugging ONLY
	)
	flag.StringVar(&settingsFile, "settings-file", "", "File to recover the persistent settings.")
	flag.StringVar(&sessionFile, "session-file", "", "[DEBUG ONLY] File to save and recover the last-known session.")
	flag.Parse()

	// Parse settings file
	settingsData, useVerboseOutput := settings.SetupSettings(settingsFile)
	VERBOSE_OUTPUT = useVerboseOutput
	SetupSessions(sessionFile)

	// Log settings
	out, err := json.Marshal(settingsData)
	if err != nil {
		panic(err)
	}
	MainStatus.Log(status.OK(), "Using settings: "+string(out))

	// Init session tracking (with or without Influx)
	// Fetch and append old session from disk if allowed

	// Parse through config if found in settings file
	config, ok := settingsData["CONFIG"]
	if ok {

		// Set up timezone
		timezoneLocation, usingTimezone := config["TIMEZONE"]
		if usingTimezone {
			loc, err := time.LoadLocation(timezoneLocation)
			if err == nil {
				TIMEZONE = loc
			} else {
				// If timezone has errored
				TIMEZONE, _ = time.LoadLocation("UTC")
			}
		} else {
			// If timezone is not set in config
			TIMEZONE, _ = time.LoadLocation("UTC")
		}

		// Set up InfluxDB time series logging
		databaseHost, usingDatabase := config["CORE_DATABASE_HOST"]
		if usingDatabase {
			DB = influx.Influx{Host: databaseHost, Database: config["CORE_DATABASE_NAME"]}
			DATABASE_ENABLED = true

			// Set up ping functionality
			// Proprietary pinging for component tracking
			if config["CORE_PING_HOST"] != "" {
				status.RemotePingAddress = config["CORE_PING_HOST"]
			} else {
				MainStatus.Log(status.OK(), "[DISABLED] Not forwarding pings to host")
			}

		} else {
			DATABASE_ENABLED = false
			MainStatus.Log(status.OK(), "[DISABLED] Not logging to influx db")
		}

		// Set up bluetooth
		bluetoothAddress, usingBluetooth := config["BLUETOOTH_ADDRESS"]
		if usingBluetooth {
			bluetooth.EnableAutoRefresh()
			bluetooth.SetAddress(bluetoothAddress)
		}

		return config["DEBUG_SESSION_LOG"]
	} else {
		MainStatus.Log(status.Warning(), "No config found in settings file, not parsing through config")
		return ""
	}
}
