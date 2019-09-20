package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
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
	MainStatus.Log(logging.OK(), "Stopping MDroid Service")
	os.Exit(0)
}

// Reboot the machine
func rebootMDroid(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("OK")
	exec.Command("reboot", "now")
}

// Shutdown the current machine
func shutdownMDroid(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("OK")
	exec.Command("poweroff", "now")
}

// Send a command to a network machine, using a simple python server to recieve
func commandNetworkMachine(name string, command string) {
	machineServiceAddress, err := settings.GetSettingByName(formatting.FormatName(name), "ADDRESS")

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
