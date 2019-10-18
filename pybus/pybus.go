// Package pybus interfaces between MDroid-Core and the pyBus programs
package pybus

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/format"
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
	} else if params["command"] != "" {
		// Some commands need special timing functions
		go PushQueue(params["command"])
	} else {
		format.WriteResponse(&w, r, format.JSONResponse{Output: "Invalid command", OK: false})
		return
	}
	format.WriteResponse(&w, r, format.JSONResponse{Output: "OK", OK: true})
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
		format.WriteResponse(&w, r, format.JSONResponse{Output: "Error: One or more required params is empty", OK: false})
		return
	}

	// Format similarly to the rest of MDroid suite, removing plurals
	// Formatting allows for fuzzier requests
	device := strings.TrimSuffix(format.Name(params["device"]), "S")
	command := strings.TrimSuffix(format.Name(params["command"]), "S")

	// Parse command into a bool, make either "on" or "off" effectively
	isPositive, err := format.IsPositiveRequest(command)
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
	case "CLOWN", "NOSE":
		PushQueue("turnOnClownNose")
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
		case "MODE":
			PushQueue("pressMode")
		case "NUM":
			PushQueue("pressNumPad")
		case "1":
			PushQueue("press1")
		case "2":
			PushQueue("press2")
		case "3":
			PushQueue("press3")
		case "4":
			PushQueue("press4")
		case "5":
			PushQueue("press5")
		case "6":
			PushQueue("press6")
		default:
			PushQueue("pressStereoPower")
		}
	case "LUCIO", "CAMERA", "BOARD":
		if format.Name(command) == "AUTO" {
			settings.Set("BOARD", "POWER", "AUTO")
		} else if isPositive {
			settings.Set("BOARD", "POWER", "ON")
			mserial.Push(mserial.Writer, "powerOnBoard")
		} else {
			settings.Set("BOARD", "POWER", "OFF")
			mserial.Push(mserial.Writer, "powerOffBoard")
		}
	case "BRIGHTWING", "LTE":
		if format.Name(command) == "AUTO" {
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
		response := format.JSONResponse{Output: fmt.Sprintf("Invalid device %s", device), OK: false}
		format.WriteResponse(&w, r, response)
		return
	}

	// Yay
	format.WriteResponse(&w, r, format.JSONResponse{Output: device, OK: true})
}
