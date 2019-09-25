package main

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/gorilla/mux"
)

// MainStatus will control logging and reporting of status / warnings / errors
var MainStatus = logging.NewStatus("Main")

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
	MainStatus.Log(logging.OK(), "Stopping MDroid Service as per request")
	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "OK", Status: "success", OK: true})
	os.Exit(0)
}

// Reboot the machine
func handleReboot(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	machine, ok := params["machine"]

	if !ok {
		json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "Machine name required", Status: "fail", OK: false})
		return
	}

	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "OK", Status: "success", OK: true})
	commandNetworkMachine(machine, "reboot")
}

// Shutdown the current machine
func handleShutdown(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	machine, ok := params["machine"]

	if !ok {
		json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "Machine name required", Status: "fail", OK: false})
		return
	}

	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "OK", Status: "success", OK: true})
	commandNetworkMachine(machine, "shutdown")
}

// welcomeRoute intros MDroid-Core, proving port and service works
func welcomeRoute(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "Welcome to MDroid! This port is fully operational, see the docs for applicable routes.", Status: "success", OK: true})
}
