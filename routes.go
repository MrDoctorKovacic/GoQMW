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

// A list of pre-approved routes to PyBus for easier routing
// These GET requests can be used instead of knowing the implementation function in pybus
func sendPybusCommand(w http.ResponseWriter, r *http.Request) {
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
		// Something fishy is amok, get out of here
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
		if isPositive {
			pybus.QueuePybus("toggleDoorLocks")
		} else {
			pybus.QueuePybus("toggleDoorLocks")
		}
	case "WINDOWS":
		fallthrough
	case "WINDOW":
		if isPositive {
			pybus.QueuePybus("popWindowsUp")
		} else {
			pybus.QueuePybus("popWindowsDown")
		}
	case "CONVERTIBLE_TOP":
		fallthrough
	case "TOP":
		if isPositive {
			pybus.QueuePybus("convertibleTopUp")
		} else {
			pybus.QueuePybus("convertibleTopDown")
		}
	case "TRUNK":
		pybus.QueuePybus("openTrunk")
	case "FLASHERS":
		pybus.QueuePybus("turnOnFlashers")
	case "HAZARDS":
		if isPositive {
			pybus.QueuePybus("turnOnHazards")
		} else {
			pybus.QueuePybus("turnOnHazards")
		}
	case "INTERIOR":
		if isPositive {
			pybus.QueuePybus("interiorLightsOff")
		} else {
			pybus.QueuePybus("interiorLightsOn")
		}
	case "MODE":
		pybus.QueuePybus("pressMode")
	case "STEREO":
		fallthrough
	case "RADIO":
		pybus.QueuePybus("pressStereoPower")
	default:
		pybus.PybusStatus.Log(status.Error(), "Invalid device "+device)
		json.NewEncoder(w).Encode("ERROR: INVALID DEVICE")
		return
	}

	// Yay
	json.NewEncoder(w).Encode("OK")
}

// **
// end router functions
// **

// Configures routes, starts router with optional middleware if configured
func startRouter(debugSessionLog string) {
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
	router.HandleFunc("/session", GetSession).Methods("GET")
	router.HandleFunc("/session/socket", GetSessionSocket).Methods("GET")
	router.HandleFunc("/session/gps", GetGPSValue).Methods("GET")
	router.HandleFunc("/session/gps", SetGPSValue).Methods("POST")
	router.HandleFunc("/session/{name}", GetSessionValue).Methods("GET")
	router.HandleFunc("/session/{name}", SetSessionValue).Methods("POST")

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
	router.HandleFunc("/{device}/{command}", sendPybusCommand).Methods("GET")
	router.HandleFunc("/", welcomeRoute).Methods("GET")

	// Log all routes for debugging later, if enabled
	// The locks here slow things down, should only be used to generate a run file, not in production
	if debugSessionLog != "" {
		enabled, err := EnableLogging(debugSessionLog)
		if enabled {
			router.Use(LoggingMiddleware)
		} else {
			MainStatus.Log(status.Error(), "Failed to open debug file, is it writable?")
			MainStatus.Log(status.Error(), err.Error())
		}
	}

	log.Fatal(http.ListenAndServe(":5353", router))
}
