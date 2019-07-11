package main

import (
	"encoding/json"
	"flag"
	"strconv"

	"github.com/MrDoctorKovacic/MDroid-Core/external/bluetooth"
	"github.com/MrDoctorKovacic/MDroid-Core/external/influx"
	"github.com/MrDoctorKovacic/MDroid-Core/external/status"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
)

// Config controls program settings and general persistent settings
type Config struct {
	DatabaseHost     string
	DatabaseName     string
	BluetoothAddress string
	PingHost         string
	DebugSessionFile string
}

// MainStatus will control logging and reporting of status / warnings / errors
var MainStatus = status.NewStatus("Main")

// Configure verbose output for code in package main
var VERBOSE_OUTPUT bool

// define our router and subsequent routes here
func main() {

	// Start with program arguments
	var (
		settingsFile string
		sessionFile  string // This is for debugging ONLY
	)
	flag.StringVar(&settingsFile, "settings-file", "", "File to recover the persistent settings.")
	flag.StringVar(&sessionFile, "session-file", "", "[DEBUG ONLY] File to save and recover the last-known session.")
	flag.Parse()

	// Parse settings file
	settingsData := settings.Setup(settingsFile)

	// Log settings
	out, err := json.Marshal(settingsData)
	if err != nil {
		panic(err)
	}
	MainStatus.Log(status.OK(), "Using settings: "+string(out))

	// Init session tracking (with or without Influx)
	sessions.Setup(sessionFile)

	// Parse through config if found in settings file
	config, ok := settingsData["CONFIG"]
	if ok {

		// Set up InfluxDB time series logging
		databaseHost, usingDatabase := config["CORE_DATABASE_HOST"]
		if usingDatabase {
			DB := influx.Influx{Host: databaseHost, Database: config["CORE_DATABASE_NAME"]}

			//
			// Check if we're configed to verbose output
			//
			var verboseOutputInt int
			verboseOutput, ok := config["VERBOSE_OUTPUT"]
			if !ok {
				verboseOutputInt = 0
			} else {
				verboseOutputInt, err = strconv.Atoi(verboseOutput)
				if err != nil {
					verboseOutputInt = 0
				}
			}

			//
			// Pass DB pool and verbosity to imports
			//
			VERBOSE_OUTPUT = verboseOutputInt != 0
			settings.SetupDatabase(DB, VERBOSE_OUTPUT)
			sessions.SetupDatabase(DB, VERBOSE_OUTPUT)

			// Set up ping functionality
			// Proprietary pinging for component tracking
			if config["CORE_PING_HOST"] != "" {
				status.RemotePingAddress = config["CORE_PING_HOST"]
			} else {
				MainStatus.Log(status.OK(), "[DISABLED] Not forwarding pings to host")
			}

		} else {
			MainStatus.Log(status.OK(), "[DISABLED] Not logging to influx db")
		}

		// Set up bluetooth
		bluetoothAddress, usingBluetooth := config["BLUETOOTH_ADDRESS"]
		if usingBluetooth {
			bluetooth.EnableAutoRefresh()
			bluetooth.SetAddress(bluetoothAddress)
		}
	} else {
		MainStatus.Log(status.Warning(), "No config found in settings file, not parsing through config")
	}

	// Define routes and begin routing
	startRouter(config["DEBUG_SESSION_LOG"])
}
