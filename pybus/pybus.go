// Package pybus interfaces between MDroid-Core and the pyBus programs
package pybus

import (
	"encoding/json"
	"fmt"
	"net/http"

	"../logging"
	"github.com/gorilla/mux"
)

// PybusStatus will control logging and reporting of status / warnings / errors
var PybusStatus = logging.NewStatus("Pybus")

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
		PybusStatus.Log(logging.Error(), fmt.Sprintf("Failed to request %s from pybus: \n %s", command, err.Error()))
		return
	}
	defer resp.Body.Close()

	PybusStatus.Log(logging.OK(), fmt.Sprintf("Added %s to the Pybus Queue", command))
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
