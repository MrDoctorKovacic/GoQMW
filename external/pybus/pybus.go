package pybus

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/external/status"
	"github.com/gorilla/mux"
	zmq "github.com/pebbe/zmq4"
)

// Registered pybusRoutines, to make sure we're not calling something that doesn't exist
// (or isn't allowed...)
var pybusRoutines = make(map[string]bool, 0)

// PybusStatus will control logging and reporting of status / warnings / errors
var PybusStatus = status.NewStatus("Pybus")

// SendPyBus queries a (hopefully) running pyBus program to run a directive
func SendPyBus(msg string) {
	context, _ := zmq.NewContext()
	socket, _ := context.NewSocket(zmq.REQ)
	defer socket.Close()

	PybusStatus.Log(status.OK(), "Connecting to pyBus ZMQ Server at localhost:4884")
	socket.Connect("tcp://localhost:4884")

	// Send command
	socket.Send(msg, 0)
	PybusStatus.Log(status.OK(), "Sending PyBus command: "+msg)

	// Wait for reply:
	reply, _ := socket.Recv(0)
	PybusStatus.Log(status.OK(), "Received: "+string(reply))
}

// rollWindowsUp sends popWindowsUp 3 consecutive times
func rollWindowsUp() {
	SendPyBus("popWindowsUp")
	time.Sleep(1200 * time.Millisecond)
	SendPyBus("popWindowsUp")
}

// rollWindowsDown sends popWindowsDown 3 consecutive times
func rollWindowsDown() {
	SendPyBus("popWindowsDown")
	time.Sleep(1500 * time.Millisecond)
	SendPyBus("popWindowsDown")
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

	if params["command"] != "" {
		// Ensure command exists in routines map, and that it's currently enabled
		if _, ok := pybusRoutines[params["command"]]; !ok || !pybusRoutines[params["command"]] {
			json.NewEncoder(w).Encode(params["command"] + " is not allowed.")
			return
		}

		// Some commands need special timing functions
		switch params["command"] {
		case "rollWindowsUp":
			go rollWindowsUp()
		case "rollWindowsDown":
			go rollWindowsDown()
		default:
			go SendPyBus(params["command"])
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
