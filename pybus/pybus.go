package pybus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"sync"

	"github.com/MrDoctorKovacic/MDroid-Core/utils"
	"github.com/gorilla/mux"
)

// Queue that the PyBus program will fetch from repeatedly
var pybusQueue []string
var pybusQueueLock sync.Mutex

// PybusStatus will control logging and reporting of status / warnings / errors
var PybusStatus = utils.NewStatus("Pybus")

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

	pybusQueueLock.Lock()
	pybusQueue = append(pybusQueue, command)
	pybusQueueLock.Unlock()

	PybusStatus.Log(utils.OK(), fmt.Sprintf("Added %s to the Pybus Queue", command))
}

// PopQueue pops a directive off the queue after confirming it occured
func PopQueue(w http.ResponseWriter, r *http.Request) {
	if len(pybusQueue) > 0 {
		PybusStatus.Log(utils.OK(), fmt.Sprintf("Dumping %s from Pybus queue to get request", pybusQueue[0]))
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
		go PushQueue(params["command"])
		json.NewEncoder(w).Encode("OK")
	} else {
		json.NewEncoder(w).Encode("Invalid command")
	}
}

// RestartService will attempt to restart the pybus service
func RestartService(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("/home/pi/le/auto/pyBus/startup_pybus.sh").Output()

	if err != nil {
		PybusStatus.Log(utils.Error(), fmt.Sprintf("Error restarting PyBus: \n%s", err.Error()))
		json.NewEncoder(w).Encode(err)
	} else {
		json.NewEncoder(w).Encode(out)
	}
}
