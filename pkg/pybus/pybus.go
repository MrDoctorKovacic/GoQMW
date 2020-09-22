// Package pybus interfaces between MDroid-Core and the pyBus programs
package pybus

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/internal/core"
	"github.com/qcasey/MDroid-Core/internal/core/settings"
	"github.com/qcasey/MDroid-Core/pkg/mserial"
	"github.com/rs/zerolog/log"
)

// Setup parses this module's implementation
func Setup(router *mux.Router) {
	// Set up pybus repeat commands
	go func() {
		time.Sleep(500)
		if settings.Data.IsSet("mdroid.pybus_device") {
			runStartup()
			startRepeats()
		}
	}()

	//
	// PyBus Routes
	//
	router.HandleFunc("/pybus/{src}/{dest}/{data}/{checksum}", StartRoutine).Methods("POST")
	router.HandleFunc("/pybus/{src}/{dest}/{data}", StartRoutine).Methods("POST")
	router.HandleFunc("/pybus/{command}/{checksum}", StartRoutine).Methods("GET")
	router.HandleFunc("/pybus/{command}", StartRoutine).Methods("GET")

	//
	// Catch-Alls for (hopefully) a pre-approved pybus function
	// i.e. /doors/lock
	//
	router.HandleFunc("/{device}/{command}", ParseCommand).Methods("GET")
}

// IsPositiveRequest helps translate UP or LOCK into true or false
func isPositiveRequest(request string) (bool, error) {
	switch request {
	case "ON", "UP", "LOCK", "OPEN", "TOGGLE", "PUSH":
		return true, nil
	case "OFF", "DOWN", "UNLOCK", "CLOSE":
		return false, nil
	}

	// Command didn't match any of the above, get out of here
	return false, fmt.Errorf("Error: %s is an invalid command", request)
}

// startRepeats that will send a command only on ACC power
func startRepeats() {
	go repeatCommand("requestIgnitionStatus", 10)
	go repeatCommand("requestLampStatus", 20)
	go repeatCommand("requestVehicleStatus", 30)
	go repeatCommand("requestOdometer", 45)
	go repeatCommand("requestTimeStatus", 60)
	go repeatCommand("requestTemperatureStatus", 120)
}

// runStartup queues the startup scripts to gather initial data from PyBus
func runStartup() {
	waitUntilOnline()
	go PushQueue("requestIgnitionStatus")
	go PushQueue("requestLampStatus")
	go PushQueue("requestVehicleStatus")
	go PushQueue("requestOdometer")
	go PushQueue("requestTimeStatus")
	go PushQueue("requestTemperatureStatus")

	// Get status of door locks by quickly toggling them
	/*
		go func() {
			err := mserial.AwaitText("toggleDoorLocks")
			if err != nil {
				log.Error().Msg(err.Error())
			}
			err = mserial.AwaitText("toggleDoorLocks")
			if err != nil {
				log.Error().Msg(err.Error())
			}
		}()*/
}

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
		log.Error().Msgf("Failed to request %s from pybus: \n %s", command, err.Error())
		return
	}
	defer resp.Body.Close()

	log.Debug().Msgf("Added %s to the Pybus Queue", command)
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
		core.WriteNewResponse(&w, r, core.JSONResponse{Output: "Invalid command", OK: false})
		return
	}
	core.WriteNewResponse(&w, r, core.JSONResponse{Output: "OK", OK: true})
}

// repeatCommand endlessly, helps with request functions
func repeatCommand(command string, sleepSeconds int) {
	log.Info().Msgf("Running Pybus command %s every %d seconds", command, sleepSeconds)
	for {
		// Only push repeated pybus commands when powered, otherwise the car won't sleep
		if core.Sessions.GetBool("acc_power.value") {
			PushQueue(command)
		}
		time.Sleep(time.Duration(sleepSeconds) * time.Second)
	}
}

func waitUntilOnline() {
	log.Info().Msg("Waiting for pybus to come online...")
	for {
		if _, err := http.Get("http://localhost:8080/requestIgnitionStatus"); err == nil {
			break
		}
		time.Sleep(time.Millisecond * 100)
	}
}

// ParseCommand is a list of pre-approved routes to PyBus for easier routing
// These GET requests can be used instead of knowing the implementation function in pybus
// and are actually preferred, since we can handle strange cases
func ParseCommand(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	if len(params["device"]) == 0 || len(params["command"]) == 0 {
		core.WriteNewResponse(&w, r, core.JSONResponse{Output: "Error: One or more required params is empty", OK: false})
		return
	}

	// Format similarly to the rest of MDroid suite, removing plurals
	// Formatting allows for fuzzier requests
	device := strings.TrimSuffix(core.FormatName(params["device"]), "S")
	command := strings.TrimSuffix(core.FormatName(params["command"]), "S")

	// Parse command into a bool, make either "on" or "off" effectively
	isPositive, err := isPositiveRequest(command)
	cannotBeParsedIntoBoolean := err != nil

	// Check if we care that the request isn't formatted into an "on" or "off"
	if cannotBeParsedIntoBoolean {
		switch device {
		case "DOOR", "TOP", "CONVERTIBLE_TOP", "HAZARD", "FLASHER", "INTERIOR":
			log.Error().Msg(err.Error())
			return
		}
	}

	log.Info().Msgf("Attempting to send command %s to device %s", command, device)

	// If the car's ACC power isn't on, it won't be ready for requests. Wake it up first
	if !core.Sessions.GetBool("acc_power.value") {
		PushQueue("requestVehicleStatus") // this will be swallowed
	}

	// All I wanted was a moment or two
	// To see if you could do that switch-a-roo
	switch device {
	case "DOOR":
		doorStatus := core.Sessions.GetString("doors_locked.value")
		if mserial.Writer != nil &&
			((isPositive && doorStatus == "FALSE") || (!isPositive && doorStatus == "TRUE")) {
			mserial.PushText("toggleDoorLocks")
		} else {
			log.Info().Msgf("Request to %s doors denied, door status is %s", command, doorStatus)
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
	default:
		log.Error().Msgf("Invalid device %s", device)
		response := core.JSONResponse{Output: fmt.Sprintf("Invalid device %s", device), OK: false}
		response.Write(&w, r)
		return
	}

	// Yay
	core.WriteNewResponse(&w, r, core.JSONResponse{Output: device, OK: true})
}
