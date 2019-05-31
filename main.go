package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os/exec"

	"github.com/MrDoctorKovacic/MDroid-Core/external/bluetooth"
	"github.com/MrDoctorKovacic/MDroid-Core/external/pybus"
	"github.com/MrDoctorKovacic/MDroid-Core/external/status"
	"github.com/MrDoctorKovacic/MDroid-Core/influx"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/gorilla/mux"
)

// Config controls program settings and general persistent settings
type Config struct {
	DatabaseHost     string
	DatabaseName     string
	BluetoothAddress string
	PingHost         string
}

// MainStatus will control logging and reporting of status / warnings / errors
var MainStatus = status.NewStatus("Main")

// Reboot the machine
func reboot(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("OK")
	exec.Command("reboot", "now")
}

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
		databaseHost, usingDatabase := config["DATABASE_HOST"]
		if usingDatabase {
			DB := influx.Influx{Host: databaseHost, Database: config["DATABASE_NAME"]}

			//
			// Pass DB pool to imports
			//
			settings.SetupDatabase(DB)
			sessions.SetupDatabase(DB)

			// Set up ping functionality
			// Proprietary pinging for component tracking
			if config["PING_HOST"] != "" {
				status.RemotePingAddress = config["PING_HOST"]
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

	// Init router
	router := mux.NewRouter()

	//
	// Main routes
	//
	router.HandleFunc("/restart", reboot).Methods("GET")

	//
	// Ping routes
	//
	router.HandleFunc("/ping/{device}", status.Ping).Methods("POST")

	//
	// Session routes
	//
	router.HandleFunc("/session", sessions.GetSession).Methods("GET")
	router.HandleFunc("/session/socket", sessions.GetSessionSocket).Methods("GET")
	router.HandleFunc("/session/gps", sessions.GetGPSValue).Methods("GET")
	router.HandleFunc("/session/gps", sessions.SetGPSValue).Methods("POST")
	router.HandleFunc("/session/{name}", sessions.GetSessionValue).Methods("GET")
	router.HandleFunc("/session/{name}", sessions.SetSessionValue).Methods("POST")

	//
	// Settings routes
	//
	router.HandleFunc("/settings", settings.GetAllSettings).Methods("GET")
	router.HandleFunc("/settings/{component}", settings.GetSetting).Methods("GET")
	router.HandleFunc("/settings/{component}/{name}", settings.GetSettingValue).Methods("GET")
	router.HandleFunc("/settings/{component}/{name}/{value}", settings.SetSettingValue).Methods("POST")

	//
	// PyBus Routes
	//
	router.HandleFunc("/pybus", pybus.GetPybusRoutines).Methods("GET")
	router.HandleFunc("/pybus/queue", pybus.SendPybus).Methods("GET")
	router.HandleFunc("/pybus/restart", pybus.RestartService).Methods("GET")
	router.HandleFunc("/pybus/{src}/{dest}/{data}", pybus.StartPybusRoutine).Methods("POST")
	router.HandleFunc("/pybus/{command}", pybus.RegisterPybusRoutine).Methods("POST")
	router.HandleFunc("/pybus/{command}", pybus.StartPybusRoutine).Methods("GET")

	//
	// ALPR Routes
	//
	router.HandleFunc("/alpr/restart", sessions.RestartALPR).Methods("GET")
	router.HandleFunc("/alpr/{plate}", sessions.LogALPR).Methods("POST")

	//
	// Bluetooth routes
	//
	router.HandleFunc("/bluetooth", bluetooth.GetDeviceInfo).Methods("GET")
	router.HandleFunc("/bluetooth/getDeviceInfo", bluetooth.GetDeviceInfo).Methods("GET")
	router.HandleFunc("/bluetooth/getMediaInfo", bluetooth.GetMediaInfo).Methods("GET")
	router.HandleFunc("/bluetooth/connect", bluetooth.Connect).Methods("GET")
	router.HandleFunc("/bluetooth/prev", bluetooth.Prev).Methods("GET")
	router.HandleFunc("/bluetooth/next", bluetooth.Next).Methods("GET")
	router.HandleFunc("/bluetooth/pause", bluetooth.Pause).Methods("GET")
	router.HandleFunc("/bluetooth/play", bluetooth.Play).Methods("GET")
	router.HandleFunc("/bluetooth/refresh", bluetooth.ForceRefresh).Methods("GET")

	// Status Routes
	router.HandleFunc("/status", status.GetStatus).Methods("GET")
	router.HandleFunc("/status/{name}", status.GetStatusValue).Methods("GET")
	router.HandleFunc("/status/{name}", status.SetStatus).Methods("POST")

	log.Fatal(http.ListenAndServe(":5353", router))
}
