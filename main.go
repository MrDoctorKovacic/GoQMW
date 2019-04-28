package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/MrDoctorKovacic/MDroid-Core/external/bluetooth"
	"github.com/MrDoctorKovacic/MDroid-Core/external/ping"
	"github.com/MrDoctorKovacic/MDroid-Core/external/pybus"
	"github.com/MrDoctorKovacic/MDroid-Core/external/status"
	"github.com/MrDoctorKovacic/MDroid-Core/external/streams"
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
	SettingsFile     string
}

// MainStatus will control logging and reporting of status / warnings / errors
var MainStatus = status.NewStatus("Main")

// Reboot the machine
func reboot(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("OK")
	exec.Command("reboot", "now")
}

// parseConfig will open and interpret program settings,
// as well as return the generic settings from last session
func parseConfig(configFile string) Config {
	// Open settings file
	file, _ := os.Open(configFile)
	defer file.Close()
	decoder := json.NewDecoder(file)
	config := Config{}
	err := decoder.Decode(&config)
	if err != nil {
		MainStatus.Log(status.OK(), "Error parsing config file '"+configFile+"': "+err.Error())
	}

	// Log our success
	MainStatus.Log(status.OK(), "Parsed config file '"+configFile+"'")

	return config
}

// define our router and subsequent routes here
func main() {

	// Start with program arguments
	var (
		configFile  string
		sessionFile string // This is for debugging ONLY
	)
	flag.StringVar(&configFile, "config-file", "", "File to recover the last-known settings.")
	flag.StringVar(&sessionFile, "session-file", "", "[DEBUG ONLY] File to save and recover the last-known session.")
	flag.Parse()

	// Parse settings file
	config := parseConfig(configFile)

	// Log config
	out, err := json.Marshal(config)
	if err != nil {
		panic(err)
	}
	MainStatus.Log(status.OK(), "Using config: "+string(out))

	// Pass settings to be interpreted
	settings.Setup(config.SettingsFile)

	// Init session tracking (with or without Influx)
	sessions.Setup(sessionFile)

	if config.DatabaseHost != "" {
		DB := influx.Influx{Host: config.DatabaseHost, Database: config.DatabaseName}

		//
		// Pass DB pool to imports
		//
		settings.SetupDatabase(DB)
		sessions.SetupDatabase(DB)

		// Proprietary pinging for component tracking
		if config.PingHost != "" {
			ping.Setup(DB, config.PingHost)
		}
	} else {
		log.Println("[Config] Not logging to influx db.")
	}

	// Pass argument to its rightful owner
	bluetooth.SetAddress(config.BluetoothAddress)

	// Init router
	router := mux.NewRouter()

	//
	// Main routes
	//
	router.HandleFunc("/restart", reboot).Methods("GET")

	//
	// Ping routes
	//
	router.HandleFunc("/ping/{device}", ping.Ping).Methods("POST")

	//
	// Session routes
	//
	router.HandleFunc("/session", sessions.GetSession).Methods("GET")
	router.HandleFunc("/session/{name}", sessions.GetSessionValue).Methods("GET")
	router.HandleFunc("/session/gps", sessions.SetGPSValue).Methods("POST")
	router.HandleFunc("/session/{name}", sessions.SetSessionValue).Methods("POST")

	//
	// Settings routes
	//
	router.HandleFunc("/settings", settings.GetAllSettings).Methods("GET")
	router.HandleFunc("/settings/{component}", settings.GetSetting).Methods("GET")
	router.HandleFunc("/settings/{component}/{name}", settings.GetSettingValue).Methods("GET")
	router.HandleFunc("/settings/{component}/{name}/{value}", settings.SetSettingValue).Methods("POST")

	//
	// Stream Routes
	//
	router.HandleFunc("/streams", streams.GetAllStreams).Methods("GET")
	router.HandleFunc("/streams/{name}", streams.GetStream).Methods("GET")
	router.HandleFunc("/streams/{name}", streams.RegisterStream).Methods("POST")
	router.HandleFunc("/streams/{name}/restart", streams.RestartStream).Methods("GET")

	//
	// PyBus Routes
	//
	router.HandleFunc("/pybus/{command}", pybus.RegisterPybusRoutine).Methods("POST")
	router.HandleFunc("/pybus/{command}", pybus.StartPybusRoutine).Methods("GET")
	router.HandleFunc("/pybus/queue", pybus.SendPybus).Methods("GET")
	router.HandleFunc("/pybus/restart", pybus.RestartService).Methods("GET")

	//
	// ALPR Routes
	//
	router.HandleFunc("/alpr/{plate}", sessions.LogALPR).Methods("POST")
	router.HandleFunc("/alpr/restart", sessions.RestartALPR).Methods("GET")

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
	router.HandleFunc("/bluetooth/restart", bluetooth.RestartService).Methods("GET")

	// Status Routes
	router.HandleFunc("/status", status.GetStatus).Methods("GET")
	router.HandleFunc("/status/{name}", status.GetStatusValue).Methods("GET")
	router.HandleFunc("/status/{name}", status.SetStatus).Methods("POST")

	log.Fatal(http.ListenAndServe(":5353", router))
}
