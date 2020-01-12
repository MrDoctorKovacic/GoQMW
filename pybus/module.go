package pybus

import (
	"time"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/pybus"
)

// Module begins module init
type Module struct{}

// Mod exports our Module implementation
var Mod *Module

// Setup parses this module's implementation
func (*Module) Setup(configAddr *map[string]string) {
	configMap := *configAddr

	// Set up pybus repeat commands
	go func() {
		time.Sleep(500)
		if _, usingPybus := configMap["PYBUS_DEVICE"]; usingPybus {
			runStartup()
			startRepeats()
		}
	}()
}

// SetRoutes implements MDroid module functions
func SetRoutes(router *mux.Router) {
	//
	// PyBus Routes
	//
	router.HandleFunc("/pybus/{src}/{dest}/{data}/{checksum}", pybus.StartRoutine).Methods("POST")
	router.HandleFunc("/pybus/{src}/{dest}/{data}", pybus.StartRoutine).Methods("POST")
	router.HandleFunc("/pybus/{command}/{checksum}", pybus.StartRoutine).Methods("GET")
	router.HandleFunc("/pybus/{command}", pybus.StartRoutine).Methods("GET")

	//
	// Catch-Alls for (hopefully) a pre-approved pybus function
	// i.e. /doors/lock
	//
	router.HandleFunc("/{device}/{command}", pybus.ParseCommand).Methods("GET")
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
