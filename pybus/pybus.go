// Package pybus interfaces between MDroid-Core and the pyBus programs
package pybus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// PushQueue adds a directive to the pybus queue
// msg can either be a directive (e.g. 'openTrunk')
// or a Python formatted list of three byte strings: src, dest, and data
// e.g. '["50", "68", "3B01"]'
func PushQueue(command string) {

	//
	// First, interrupt with some special cases
	//
	switch command {
	case "rollWindowsUp":
		go PushQueue("popWindowsUp")
		go PushQueue("popWindowsUp")
		return
	case "rollWindowsDown":
		go PushQueue("popWindowsDown")
		go PushQueue("popWindowsDown")
		return
	}

	// Send request to pybus server
	resp, err := http.Get(fmt.Sprintf("http://localhost:8080/%s", command))
	if err != nil {
		log.Error().Msg(fmt.Sprintf("Failed to request %s from pybus: \n %s", command, err.Error()))
		return
	}
	defer resp.Body.Close()

	log.Info().Msg(fmt.Sprintf("Added %s to the Pybus Queue", command))
}

// StartRoutine handles incoming requests to the pybus program, will add routines to the queue
func StartRoutine(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	src, srcOK := params["src"]
	dest, destOK := params["dest"]
	data, dataOK := params["data"]

	if srcOK && destOK && dataOK && len(src) == 2 && len(dest) == 2 && len(data) > 0 {
		go PushQueue(fmt.Sprintf(`["%s", "%s", "%s"]`, src, dest, data))
		json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "OK", Status: "success", OK: true})
	} else if params["command"] != "" {
		// Some commands need special timing functions
		go PushQueue(params["command"])
		json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "OK", Status: "success", OK: true})
	} else {
		json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "Invalid command", Status: "fail", OK: false})
	}
}

// RepeatCommand endlessly, helps with request functions
func RepeatCommand(command string, sleepSeconds int) {
	for {
		// Only push repeated pybus commands when powered, otherwise the car won't sleep
		if hasPower, err := sessions.Get("ACC_POWER"); err == nil && hasPower.Value == "TRUE" {
			PushQueue(command)
		}
		time.Sleep(time.Duration(sleepSeconds) * time.Second)
	}
}

// ParseCommand is a list of pre-approved routes to PyBus for easier routing
// These GET requests can be used instead of knowing the implementation function in pybus
// and are actually preferred, since we can handle strange cases
func ParseCommand(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	if len(params["device"]) == 0 || len(params["command"]) == 0 {
		json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "Error: One or more required params is empty", Status: "fail", OK: false})
		return
	}

	// Format similarly to the rest of MDroid suite, removing plurals
	// Formatting allows for fuzzier requests
	device := strings.TrimSuffix(formatting.FormatName(params["device"]), "S")
	command := strings.TrimSuffix(formatting.FormatName(params["command"]), "S")

	// Parse command into a bool, make either "on" or "off" effectively
	isPositive, err := formatting.IsPositiveRequest(command)
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	// Log if requested
	log.Info().Msg(fmt.Sprintf("Attempting to send command %s to device %s", command, device))

	// All I wanted was a moment or two to
	// See if you could do that switch-a-roo
	switch device {
	case "DOOR":
		doorStatus, _ := sessions.Get("DOORS_LOCKED")

		// Since this toggles, we should only do lock/unlock the doors if there's a known state
		/*if err != nil && !isPositive {
			log.Info().Msg("Door status is unknown, but we're locking. Go through the pybus")
			PushQueue("lockDoors")
		} else*/
		if mserial.Writer != nil && isPositive && doorStatus.Value == "FALSE" ||
			mserial.Writer != nil && !isPositive && doorStatus.Value == "TRUE" {
			mserial.Push(mserial.Writer, "toggleDoorLocks")
		}
	case "WINDOW":
		if command == "POPDOWN" {
			PushQueue("popWindowsDown")
		} else if command == "POPUP" {
			PushQueue("popWindowsUp")
		} else if isPositive {
			PushQueue("rollWindowsUp")
		} else {
			PushQueue("rollWindowsDown")
		}
	case "TOP", "CONVERTIBLE_TOP":
		if isPositive {
			PushQueue("convertibleTopUp")
		} else {
			PushQueue("convertibleTopDown")
		}
	case "TRUNK":
		PushQueue("openTrunk")
	case "HAZARD":
		if isPositive {
			PushQueue("turnOnHazards")
		} else {
			PushQueue("turnOffAllExteriorLights")
		}
	case "FLASHER":
		if isPositive {
			PushQueue("flashAllExteriorLights")
		} else {
			PushQueue("turnOffAllExteriorLights")
		}
	case "INTERIOR":
		if isPositive {
			PushQueue("interiorLightsOff")
		} else {
			PushQueue("interiorLightsOn")
		}
	case "MODE":
		PushQueue("pressMode")
	case "RADIO", "NAV", "STEREO":
		switch command {
		case "AM":
			PushQueue("pressAM")
		case "FM":
			PushQueue("pressFM")
		case "NEXT":
			PushQueue("pressNext")
		case "PREV":
			PushQueue("pressPrev")
		case "NUM":
			PushQueue("pressNumPad")
		case "MODE":
			PushQueue("pressMode")
		default:
			PushQueue("pressStereoPower")
		}
	case "LUCIO", "CAMERA", "BOARD":
		if formatting.FormatName(command) == "AUTO" {
			settings.Set("BOARD", "POWER", "AUTO")
		} else if isPositive {
			settings.Set("BOARD", "POWER", "ON")
			mserial.Push(mserial.Writer, "powerOnBoard")
		} else {
			settings.Set("BOARD", "POWER", "OFF")
			mserial.Push(mserial.Writer, "powerOffBoard")
		}
	case "BRIGHTWING", "LTE":
		if formatting.FormatName(command) == "AUTO" {
			settings.Set("WIRELESS", "POWER", "AUTO")
		} else if isPositive {
			settings.Set("WIRELESS", "POWER", "ON")
			mserial.Push(mserial.Writer, "powerOnWireless")
		} else {
			settings.Set("WIRELESS", "POWER", "OFF")
			mserial.Push(mserial.Writer, "powerOffWireless")
		}
	default:
		log.Error().Msg(fmt.Sprintf("Invalid device %s", device))
		json.NewEncoder(w).Encode(formatting.JSONResponse{Output: fmt.Sprintf("Invalid device %s", device), Status: "fail", OK: false})
		return
	}

	// Yay
	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: device, Status: "success", OK: true})
}
