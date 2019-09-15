package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/MrDoctorKovacic/MDroid-Core/bluetooth"
	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/pybus"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
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

// Shutdown the machine
func shutdown(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("OK")
	exec.Command("poweroff", "now")
}

// Stop MDroid-Core service
func stop(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("OK")
	MainStatus.Log(logging.OK(), "Stopping MDroid Service")
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

	if len(params["device"]) == 0 || len(params["command"]) == 0 {
		json.NewEncoder(w).Encode("Error: One or more required params is empty")
		return
	}

	// Format similarly to the rest of MDroid suite, removing plurals
	// Formatting allows for fuzzier requests
	device := strings.TrimSuffix(formatting.FormatName(params["device"]), "S")
	command := strings.TrimSuffix(formatting.FormatName(params["command"]), "S")

	// Parse command into a bool, make either "on" or "off" effectively
	isPositive, _ := formatting.IsPositiveRequest(command)

	// Log if requested
	pybus.PybusStatus.Log(logging.OK(), fmt.Sprintf("Attempting to send command %s to device %s", command, device))

	// All I wanted was a moment or two to
	// See if you could do that switch-a-roo
	switch device {
	case "DOOR":
		deviceStatus, err := GetSessionValue("DOORS_LOCKED")

		// Since this toggles, we should only do lock/unlock the doors if there's a known state
		if err != nil && !isPositive {
			pybus.PybusStatus.Log(logging.OK(), "Door status is unknown, but we're locking. Go through the pybus")
			pybus.PushQueue("lockDoors")
		} else {
			if Config.HardwareSerialEnabled {
				if isPositive && deviceStatus.Value == "FALSE" ||
					!isPositive && deviceStatus.Value == "TRUE" {
					WriteSerial("toggleDoorLocks")
				}
			}
		}
	case "WINDOW":
		if command == "POPDOWN" {
			pybus.PushQueue("popWindowsDown")
		} else if command == "POPUP" {
			pybus.PushQueue("popWindowsUp")
		} else if isPositive {
			pybus.PushQueue("rollWindowsUp")
		} else {
			pybus.PushQueue("rollWindowsDown")
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
	case "HAZARD":
		if isPositive {
			pybus.PushQueue("turnOnHazards")
		} else {
			pybus.PushQueue("turnOffAllExteriorLights")
		}
	case "FLASHER":
		if isPositive {
			pybus.PushQueue("flashAllExteriorLights")
		} else {
			pybus.PushQueue("turnOffAllExteriorLights")
		}
	case "INTERIOR":
		if isPositive {
			pybus.PushQueue("interiorLightsOff")
		} else {
			pybus.PushQueue("interiorLightsOn")
		}
	case "MODE":
		pybus.PushQueue("pressMode")
	case "NAV":
		fallthrough
	case "STEREO":
		fallthrough
	case "RADIO":
		if command == "AM" {
			pybus.PushQueue("pressAM")
		} else if command == "FM" {
			pybus.PushQueue("pressFM")
		} else if command == "NEXT" {
			pybus.PushQueue("pressNext")
		} else if command == "PREV" {
			pybus.PushQueue("pressPrev")
		} else if command == "NUM" {
			pybus.PushQueue("pressNumPad")
		} else if command == "MODE" {
			pybus.PushQueue("pressMode")
		} else {
			pybus.PushQueue("pressStereoPower")
		}
	case "CAMERA":
		fallthrough
	case "BOARD":
		fallthrough
	case "ARTANIS":
		if isPositive {
			WriteSerial("powerOnBoard")
		} else {
			WriteSerial("powerOffBoard")
		}
	case "LTE":
		fallthrough
	case "BRIGHTWING":
		if isPositive {
			WriteSerial("powerOnWireless")
		} else {
			WriteSerial("powerOffWireless")
		}
	default:
		pybus.PybusStatus.Log(logging.Error(), fmt.Sprintf("Invalid device %s", device))
		json.NewEncoder(w).Encode(fmt.Sprintf("Invalid device %s", device))
		return
	}

	// Yay
	json.NewEncoder(w).Encode(device)
}

func handleSlackAlert(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	if Config.SlackURL != "" {
		logging.SlackAlert(Config.SlackURL, params["message"])
	} else {
		json.NewEncoder(w).Encode("Slack URL not set in config.")
	}

	// Echo back message
	json.NewEncoder(w).Encode(params["message"])
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
	router.HandleFunc("/shutdown", shutdown).Methods("GET")
	router.HandleFunc("/stop", stop).Methods("GET")

	//
	// Ping routes
	//
	router.HandleFunc("/ping/{device}", logging.Ping).Methods("POST")

	//
	// Session routes
	//
	router.HandleFunc("/session", HandleGetSession).Methods("GET")
	router.HandleFunc("/session/socket", GetSessionSocket).Methods("GET")
	router.HandleFunc("/session/gps", GetGPSValue).Methods("GET")
	router.HandleFunc("/session/gps", SetGPSValue).Methods("POST")
	router.HandleFunc("/session/{name}", GetSessionValueHandler).Methods("GET")
	router.HandleFunc("/session/{name}", PostSessionValue).Methods("POST")

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
	router.HandleFunc("/pybus/restart", pybus.RestartService).Methods("GET")
	router.HandleFunc("/pybus/{src}/{dest}/{data}", pybus.StartRoutine).Methods("POST")
	router.HandleFunc("/pybus/{command}", pybus.StartRoutine).Methods("GET")

	//
	// Serial routes
	//
	router.HandleFunc("/serial/{command}", WriteSerialHandler).Methods("POST")

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
	router.HandleFunc("/status", logging.GetStatus).Methods("GET")
	router.HandleFunc("/status/{name}", logging.GetStatusValue).Methods("GET")
	router.HandleFunc("/status/{name}", logging.SetStatus).Methods("POST")
	router.HandleFunc("/alert/{message}", handleSlackAlert).Methods("GET")

	//
	// Catch-Alls for (hopefully) a pre-approved pybus function
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
		enabled, err := logging.EnableLogging(Config.DebugSessionFile, Timezone)
		if enabled {
			router.Use(logging.LogMiddleware)
		} else {
			MainStatus.Log(logging.Error(), "Failed to open debug file, is it writable?")
			MainStatus.Log(logging.Error(), err.Error())
		}
	}

	if Config.AuthToken != "" {
		// Ask for matching Auth Token before taking requests
		router.Use(AuthMiddleware)
	}

	log.Fatal(http.ListenAndServe(":5353", router))
}
