package pybus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"

	"github.com/MrDoctorKovacic/MDroid-Core/status"
	"github.com/gorilla/mux"
)

// Registered pybusRoutines, to make sure we're not calling something that doesn't exist
// (or isn't allowed...)
var pybusRoutines = make(map[string]bool, 0)

var pybusQueue []string

// PybusStatus will control logging and reporting of status / warnings / errors
var PybusStatus = status.NewStatus("Pybus")

// QueuePybus adds a directive to the pybus queue
func QueuePybus(msg string) {
	pybusQueue = append(pybusQueue, msg)
	PybusStatus.Log(status.OK(), "Added "+msg+" to the Pybus Queue")
}

// SendPybus pops a directive off the queue after confirming it occured
func SendPybus(w http.ResponseWriter, r *http.Request) {
	if len(pybusQueue) > 0 {
		PybusStatus.Log(status.OK(), "Dumping "+pybusQueue[0]+" from Pybus queue to get request")
		json.NewEncoder(w).Encode(pybusQueue[0])

		// Pop off queue
		pybusQueue = pybusQueue[1:]
	} else {
		json.NewEncoder(w).Encode("{}")
	}
}

// GetPybusRoutines fetches all registered Pybus routines
func GetPybusRoutines(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(pybusRoutines)
}

// RegisterPybusRoutine handles PyBus goroutine
func RegisterPybusRoutine(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	// Append new routine into mapping
	pybusRoutines[params["command"]] = true

	// Log success
	PybusStatus.Log(status.OK(), "Registered new routine: "+params["command"])
	json.NewEncoder(w).Encode("OK")
}

// DisablePybusRoutine handles PyBus goroutine
func DisablePybusRoutine(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	pybusRoutines[params["command"]] = false

	// Log success
	PybusStatus.Log(status.OK(), "Disabled routine: "+params["command"])
	json.NewEncoder(w).Encode("OK")
}

// StartPybusRoutine handles PyBus goroutine
func StartPybusRoutine(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	src, srcOK := params["src"]
	dest, destOK := params["dest"]
	data, dataOK := params["data"]

	if srcOK && destOK && dataOK && len(src) == 2 && len(dest) == 2 && len(data) > 0 {
		go QueuePybus(fmt.Sprintf(`["%s", "%s", "%s"]`, src, dest, data))
		json.NewEncoder(w).Encode("OK")
	} else if params["command"] != "" {
		// Ensure command exists in routines map, and that it's currently enabled
		/*if _, ok := pybusRoutines[params["command"]]; !ok || !pybusRoutines[params["command"]] {
			json.NewEncoder(w).Encode(params["command"] + " is not allowed.")
			return
		}*/

		// Some commands need special timing functions
		switch params["command"] {
		case "rollWindowsUp":
			QueuePybus("popWindowsUp")
			QueuePybus("popWindowsUp")
		case "rollWindowsDown":
			QueuePybus("popWindowsDown")
			QueuePybus("popWindowsDown")
		default:
			go QueuePybus(params["command"])
		}

		json.NewEncoder(w).Encode("OK")
	} else {
		json.NewEncoder(w).Encode("Invalid command")
	}
}

// RestartService will attempt to restart the pybus service
func RestartService(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("/home/pi/le/auto/pyBus/startup_pybus.sh").Output()

	if err != nil {
		PybusStatus.Log(status.Error(), "Error restarting PyBus: "+err.Error())
		json.NewEncoder(w).Encode(err)
	} else {
		json.NewEncoder(w).Encode(out)
	}
}
