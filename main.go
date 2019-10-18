package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/rs/zerolog/log"
	"github.com/tarm/serial"
)

// define our router and subsequent routes here
func main() {

	// Run through the config file and set up some global variables
	parseConfig()

	// Define routes and begin routing
	startRouter()
}

// Some shutdowns are more complicated than others, ensure we shut down safely
func gracefulShutdown(name string) {
	serialCommand := fmt.Sprintf("powerOff%s", name)

	switch name {
	case "Board":
		sendServiceCommand("etc", "shutdown")
		machineShutdown(mserial.Writer, "board", time.Second*10, serialCommand)
	case "Wireless":
		machineShutdown(mserial.Writer, "lte", time.Second*10, serialCommand)
	case "Sound":
		machineShutdown(mserial.Writer, "sound", time.Second*10, serialCommand)
	default:
		mserial.Push(mserial.Writer, serialCommand)
	}
}

// sendServiceCommand sends a command to a network machine, using a simple python server to recieve
func sendServiceCommand(name string, command string) {
	machineServiceAddress, err := settings.Get(formatting.Name(name), "ADDRESS")
	if machineServiceAddress == "" {
		return
	}

	resp, err := http.Get(fmt.Sprintf("http://%s:5350/%s", machineServiceAddress, command))
	if err != nil {
		log.Error().Msg(fmt.Sprintf("Failed to command machine %s (at %s) to %s: \n%s", name, machineServiceAddress, command, err.Error()))
		return
	}
	defer resp.Body.Close()
}

// machineShutdown shutdowns the named machine safely
func machineShutdown(serialDevice *serial.Port, machine string, timeToSleep time.Duration, serialMessage string) {
	sendServiceCommand(machine, "shutdown")
	time.Sleep(timeToSleep)
	mserial.Push(serialDevice, serialMessage)
}
