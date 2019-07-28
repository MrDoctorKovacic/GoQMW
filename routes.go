package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/MrDoctorKovacic/MDroid-Core/bluetooth"
	"github.com/MrDoctorKovacic/MDroid-Core/pybus"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/MrDoctorKovacic/MDroid-Core/status"
	"github.com/gorilla/mux"
)

// **
// Start with some router functions
// **

// Reboot the machine
func reboot(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("OK")
	exec.Command("reboot", "now")
}

// Stop MDroid-Core service
func stop(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("OK")
	MainStatus.Log(status.OK(), "Stopping MDroid Service")
	os.Exit(0)
}

// welcomeRoute intros MDroid-Core, proving port and service works
func welcomeRoute(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("Welcome to MDroid! This port is fully operational, see the docs for applicable routes.")
}

// a list of pre-approved routes to PyBus for easier routing
// These GET requests can be used instead of knowing the implementation function in pybus
// and are actually preferred, since we can handle strange cases
func parseCommand(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	// Format similarly to the rest of MDroid suite
	device := strings.TrimSpace(strings.ToUpper(strings.Replace(params["device"], " ", "_", -1)))
	commandRaw := strings.TrimSpace(strings.ToUpper(strings.Replace(params["command"], " ", "_", -1)))

	// Parse command into a bool, make either "on" or "off" effectively
	var isPositive bool
	if commandRaw == "ON" || commandRaw == "UP" || commandRaw == "LOCK" || commandRaw == "OPEN" || commandRaw == "TOGGLE" || commandRaw == "PUSH" {
		isPositive = true
	} else if commandRaw == "OFF" || commandRaw == "DOWN" || commandRaw == "UNLOCK" || commandRaw == "CLOSE" {
		isPositive = false
	} else {
		// Command didn't match any of the above, get out of here
		json.NewEncoder(w).Encode("ERROR: INVALID COMMAND")
		return
	}

	// Log if requested
	pybus.PybusStatus.Log(status.OK(), "Attempting to put "+commandRaw+" to device "+device)

	// It ain't really that hard to do and
	// I ain't trying to be in love with you and
	// All I wanted was a moment or two to
	// See if you could do that switch-a-roo
	switch device {
	case "DOORS":
		fallthrough
	case "DOOR":
		// Since this toggles, we should only do lock/unlock the doors if there's a known state
		deviceStatus, ok := GetSessionValue("DOORS_LOCKED")
		if ok && Config.HardwareSerialEnabled &&
			(isPositive && deviceStatus.Value == "FALSE") ||
			(!isPositive && deviceStatus.Value == "TRUE") {
			WriteSerial("toggleDoorLocks")
		}
	case "WINDOWS":
		fallthrough
	case "WINDOW":
		if isPositive {
			pybus.PushQueue("popWindowsUp")
			pybus.PushQueue("popWindowsUp")
		} else {
			pybus.PushQueue("popWindowsDown")
			pybus.PushQueue("popWindowsDown")
		}
	case "CONVERTIBLE_TOP":
		fallthrough
	case "TOP":
		if isPositive {
			pybus.PushQueue("convertibleTopUp")
		} else {
			pybus.PushQueue("convertibleTopDown")
		}
	case "TRUNK":
		pybus.PushQueue("openTrunk")
	case "HAZARDS":
		if Config.HardwareSerialEnabled {
			WriteSerial("toggleHazards")
		}
	case "INTERIOR":
		if isPositive {
			pybus.PushQueue("interiorLightsOff")
		} else {
			pybus.PushQueue("interiorLightsOn")
		}
	case "MODE":
		pybus.PushQueue("pressMode")
	case "STEREO":
		fallthrough
	case "RADIO":
		pybus.PushQueue("pressStereoPower")
	default:
		pybus.PybusStatus.Log(status.Error(), "Invalid device "+device)
		json.NewEncoder(w).Encode("ERROR: INVALID DEVICE")
		return
	}

	// Yay
	json.NewEncoder(w).Encode(device)
}

// **
// end router functions
// **

// Configures routes, starts router with optional middleware if configured
func startRouter() {
	// Init router
	router := mux.NewRouter()

	//
	// Main routes
	//
	router.HandleFunc("/restart", reboot).Methods("GET")
	router.HandleFunc("/stop", stop).Methods("GET")

	//
	// Ping routes
	//
	router.HandleFunc("/ping/{device}", status.Ping).Methods("POST")

	//
	// Session routes
	//
	router.HandleFunc("/session", HandleGetSession).Methods("GET")
	router.HandleFunc("/session/socket", GetSessionSocket).Methods("GET")
	router.HandleFunc("/session/gps", GetGPSValue).Methods("GET")
	router.HandleFunc("/session/gps", SetGPSValue).Methods("POST")
	router.HandleFunc("/session/{name}", HandleGetSessionValue).Methods("GET")
	router.HandleFunc("/session/{name}", HandleSetSessionValue).Methods("POST")

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
	router.HandleFunc("/pybus/queue", pybus.PopQueue).Methods("GET")
	router.HandleFunc("/pybus/restart", pybus.RestartService).Methods("GET")
	router.HandleFunc("/pybus/{src}/{dest}/{data}", pybus.StartRoutine).Methods("POST")
	router.HandleFunc("/pybus/{command}", pybus.StartRoutine).Methods("GET")

	//
	// ALPR Routes
	//
	router.HandleFunc("/alpr/restart", RestartALPR).Methods("GET")
	router.HandleFunc("/alpr/{plate}", LogALPR).Methods("POST")

	//
	// Bluetooth routes
	//
	router.HandleFunc("/bluetooth", bluetooth.GetDeviceInfo).Methods("GET")
	router.HandleFunc("/bluetooth/getDeviceInfo", bluetooth.GetDeviceInfo).Methods("GET")
	router.HandleFunc("/bluetooth/getMediaInfo", bluetooth.GetMediaInfo).Methods("GET")
	router.HandleFunc("/bluetooth/connect", bluetooth.Connect).Methods("GET")
	router.HandleFunc("/bluetooth/disconnect", bluetooth.Connect).Methods("GET")
	router.HandleFunc("/bluetooth/prev", bluetooth.Prev).Methods("GET")
	router.HandleFunc("/bluetooth/next", bluetooth.Next).Methods("GET")
	router.HandleFunc("/bluetooth/pause", bluetooth.Pause).Methods("GET")
	router.HandleFunc("/bluetooth/play", bluetooth.Play).Methods("GET")
	router.HandleFunc("/bluetooth/refresh", bluetooth.ForceRefresh).Methods("GET")

	//
	// Status Routes
	//
	router.HandleFunc("/status", status.GetStatus).Methods("GET")
	router.HandleFunc("/status/{name}", status.GetStatusValue).Methods("GET")
	router.HandleFunc("/status/{name}", status.SetStatus).Methods("POST")

	//
	// Catch-All for (hopefully) a pre-approved pybus function
	// i.e. /doors/lock
	//
	router.HandleFunc("/{device}/{command}", parseCommand).Methods("GET")

	//
	// Finally, welcome and meta routes
	//
	router.HandleFunc("/", welcomeRoute).Methods("GET")

	if Config.DebugSessionFile != "" {
		// Log all routes for debugging later, if enabled
		// The locks here slow things down, should only be used to generate a run file, not in production
		enabled, err := EnableLogging(Config.DebugSessionFile)
		if enabled {
			router.Use(LoggingMiddleware)
		} else {
			MainStatus.Log(status.Error(), "Failed to open debug file, is it writable?")
			MainStatus.Log(status.Error(), err.Error())
		}
	}

	if Config.AuthToken != "" {
		// Ask for matching Auth Token before taking requests
		router.Use(AuthMiddleware)
	}

	log.Fatal(http.ListenAndServe(":5353", router))
}
