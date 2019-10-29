package main

import (
	"fmt"
	"net/http"

	"github.com/MrDoctorKovacic/MDroid-Core/format"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/rs/zerolog/log"
)

// define our router and subsequent routes here
func main() {

	// Run through the config file and set up some global variables
	parseConfig()

	// Define routes and begin routing
	startRouter()
}

// sendServiceCommand sends a command to a network machine, using a simple python server to recieve
func sendServiceCommand(name string, command string) {
	machineServiceAddress, err := settings.Get(format.Name(name), "ADDRESS")
	if machineServiceAddress == "" {
		log.Error().Msg(fmt.Sprintf("Device %s address not found, not issuing %s", name, command))
		return
	}

	resp, err := http.Get(fmt.Sprintf("http://%s:5350/%s", machineServiceAddress, command))
	if err != nil {
		log.Error().Msg(fmt.Sprintf("Failed to command machine %s (at %s) to %s: \n%s", name, machineServiceAddress, command, err.Error()))
		return
	}
	defer resp.Body.Close()
}
