package sessions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
	"github.com/MrDoctorKovacic/MDroid-Core/pybus"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/gorilla/mux"
)

// RepeatCommand endlessly, helps with request functions
func RepeatCommand(command string, sleepSeconds int) {
	for {
		// Only push repeated pybus commands when powered, otherwise the car won't sleep
		hasPower, err := GetSessionValue("ACC_POWER")
		if err == nil && hasPower.Value == "TRUE" {
			pybus.PushQueue(command)
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
			if settings.Config.HardwareSerialEnabled {
				if isPositive && deviceStatus.Value == "FALSE" ||
					!isPositive && deviceStatus.Value == "TRUE" {
					mserial.WriteSerial(settings.Config.SerialControlDevice, "toggleDoorLocks")
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
	case "LUCIO":
		if formatting.FormatName(command) == "AUTO" {
			settings.Set("LUCIO", "POWER", "AUTO")
			return
		} else if isPositive {
			settings.Set("LUCIO", "POWER", "ON")
			mserial.WriteSerial(settings.Config.SerialControlDevice, "powerOnBoard")
		} else {
			settings.Set("LUCIO", "POWER", "OFF")
			mserial.WriteSerial(settings.Config.SerialControlDevice, "powerOffBoard")
		}
	case "LTE":
		fallthrough
	case "BRIGHTWING":
		if formatting.FormatName(command) == "AUTO" {
			settings.Set("LUCIO", "POWER", "AUTO")
			return
		}
		if isPositive {
			settings.Set("BRIGHTWING", "POWER", "ON")
			mserial.WriteSerial(settings.Config.SerialControlDevice, "powerOnWireless")
		} else {
			settings.Set("BRIGHTWING", "POWER", "OFF")
			mserial.WriteSerial(settings.Config.SerialControlDevice, "powerOffWireless")
		}
	default:
		pybus.PybusStatus.Log(logging.Error(), fmt.Sprintf("Invalid device %s", device))
		json.NewEncoder(w).Encode(formatting.JSONResponse{Output: fmt.Sprintf("Invalid device %s", device), Status: "fail", OK: false})
		return
	}

	// Yay
	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: device, Status: "success", OK: true})
}
