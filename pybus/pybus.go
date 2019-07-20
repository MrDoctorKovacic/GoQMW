package pybus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"

	"github.com/MrDoctorKovacic/MDroid-Core/status"
	"github.com/gorilla/mux"
)

// Hardware serial is a gateway to an Arduino hooked to a set of relays
var USING_HARDWARE_SERIAL bool

// Queue that the PyBus program will fetch from repeatedly
var pybusQueue []string

// PybusStatus will control logging and reporting of status / warnings / errors
var PybusStatus = status.NewStatus("Pybus")

// PushQueue adds a directive to the pybus queue
// msg can either be a directive (e.g. 'openTrunk')
// or a Python formatted list of three byte strings: src, dest, and data
// e.g. '["50", "68", "3B01"]'
func PushQueue(msg string) {

	//
	// First, interrupt with some special cases
	//
	switch msg {
	case "unlockDoors":
	}

	pybusQueue = append(pybusQueue, msg)
	PybusStatus.Log(status.OK(), "Added "+msg+" to the Pybus Queue")
}

// PopQueue pops a directive off the queue after confirming it occured
func PopQueue(w http.ResponseWriter, r *http.Request) {
	if len(pybusQueue) > 0 {
		PybusStatus.Log(status.OK(), "Dumping "+pybusQueue[0]+" from Pybus queue to get request")
		json.NewEncoder(w).Encode(pybusQueue[0])

		// Pop off queue
		pybusQueue = pybusQueue[1:]
	} else {
		json.NewEncoder(w).Encode("{}")
	}
}

// StartRoutine handles incoming requests to the pybus program, will add routines to the queue
func StartRoutine(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	src, srcOK := params["src"]
	dest, destOK := params["dest"]
	data, dataOK := params["data"]

	if srcOK && destOK && dataOK && len(src) == 2 && len(dest) == 2 && len(data) > 0 {
		go PushQueue(fmt.Sprintf(`["%s", "%s", "%s"]`, src, dest, data))
		json.NewEncoder(w).Encode("OK")
	} else if params["command"] != "" {
		// Some commands need special timing functions
		switch params["command"] {
		case "rollWindowsUp":
			PushQueue("popWindowsUp")
			PushQueue("popWindowsUp")
		case "rollWindowsDown":
			PushQueue("popWindowsDown")
			PushQueue("popWindowsDown")
		default:
			go PushQueue(params["command"])
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
