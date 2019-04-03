package pybus

import (
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/gorilla/mux"
	zmq "github.com/pebbe/zmq4"
)

// SendPyBus queries a (hopefully) running pyBus program to run a directive
func SendPyBus(msg string) {
	context, _ := zmq.NewContext()
	socket, _ := context.NewSocket(zmq.REQ)
	defer socket.Close()

	log.Printf("Connecting to pyBus ZMQ Server")
	socket.Connect("tcp://localhost:4884")

	// Send command
	socket.Send(msg, 0)
	println("Sending PyBus command: ", msg)

	// Wait for reply:
	reply, _ := socket.Recv(0)
	println("Received: ", string(reply))
}

// rollWindowsUp sends popWindowsUp 3 consecutive times
func rollWindowsUp() {
	SendPyBus("popWindowsDown")
	time.Sleep(3 * time.Second)
	SendPyBus("popWindowsDown")
	time.Sleep(3 * time.Second)
	SendPyBus("popWindowsDown")
}

// rollWindowsDown sends popWindowsDown 3 consecutive times
func rollWindowsDown() {
	SendPyBus("popWindowsDown")
	time.Sleep(3 * time.Second)
	SendPyBus("popWindowsDown")
	time.Sleep(3 * time.Second)
	SendPyBus("popWindowsDown")
}

// StartPyBusRoutine handles PyBus goroutine
func StartPyBusRoutine(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	// TODO : Get a list of available
	if params["command"] != "" {
		// Some commands need special timing functions
		switch params["command"] {
		case "rollWindowsUp":
			go rollWindowsUp()
		case "rollWindowsDown":
			go rollWindowsDown()
		default:
			go SendPyBus(params["command"])
		}

		json.NewEncoder(w).Encode(params["command"])
	}
}

// RestartService will attempt to restart the pybus service
func RestartService(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("/home/pi/le/auto/pyBus/startup_pybus.sh").Output()

	if err != nil {
		log.Println(err)
		json.NewEncoder(w).Encode(err)
	} else {
		json.NewEncoder(w).Encode(out)
	}
}
