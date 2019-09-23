package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"./formatting"
	"./logging"
	"./settings"
	"github.com/gorilla/mux"
)

// MainStatus will control logging and reporting of status / warnings / errors
var MainStatus = logging.NewStatus("Main")

// Timezone location for session last used and logging
var Timezone *time.Location

// define our router and subsequent routes here
func main() {

	// Run through the config file and set up some global variables
	parseConfig()

	// Define routes and begin routing
	startRouter()
}

// **
// Main service routes
// **

// Stop MDroid-Core service
func stopMDroid(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("OK")
	MainStatus.Log(logging.OK(), "Stopping MDroid Service as per request")
	os.Exit(0)
}

// Reboot the machine
func handleReboot(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	machine, ok := params["machine"]

	if !ok {
		json.NewEncoder(w).Encode("Machine name required")
		return
	}

	json.NewEncoder(w).Encode("OK")
	commandNetworkMachine(machine, "reboot")
}

// Shutdown the current machine
func handleShutdown(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	machine, ok := params["machine"]

	if !ok {
		json.NewEncoder(w).Encode("Machine name required")
		return
	}

	json.NewEncoder(w).Encode("OK")
	commandNetworkMachine(machine, "shutdown")
}

// Send a command to a network machine, using a simple python server to recieve
func commandNetworkMachine(name string, command string) {
	machineServiceAddress, err := settings.Get(formatting.FormatName(name), "ADDRESS")
	if machineServiceAddress == "" {
		return
	}

	resp, err := http.Get(fmt.Sprintf("http://%s:5350/%s", machineServiceAddress, command))
	if err != nil {
		MainStatus.Log(logging.Error(), fmt.Sprintf("Failed to command machine %s (at %s) to %s: \n%s", name, machineServiceAddress, command, err.Error()))
		return
	}
	defer resp.Body.Close()
}

// welcomeRoute intros MDroid-Core, proving port and service works
func welcomeRoute(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("Welcome to MDroid! This port is fully operational, see the docs for applicable routes.")
}
