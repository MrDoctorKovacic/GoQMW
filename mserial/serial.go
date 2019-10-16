package mserial

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/gorilla/mux"
	"github.com/tarm/serial"
)

// status will control logging and reporting of status / warnings / errors
var (
	status         logging.ProgramStatus
	writeQueue     map[*serial.Port][]string
	writeQueueLock sync.Mutex
)

func init() {
	status = logging.NewStatus("Serial")
	writeQueue = make(map[*serial.Port][]string, 0)
}

// WriteSerialHandler handles messages sent through the server
func WriteSerialHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	if params["command"] != "" {
		Push(settings.Config.SerialControlDevice, params["command"])
	}
	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "OK", Status: "success", OK: true})
}

// ReadSerial will continuously pull data from incoming serial
func ReadSerial(serialDevice *serial.Port, isWriter bool) (interface{}, error) {
	reader := bufio.NewReader(serialDevice)

	// While connected, try to read from the device
	// If we become disconnected, the goroutine will end and will have to be restarted
	for connected := true; connected; {
		//buf := make([]byte, 1024)
		//n, err := serialDevice.Read(buf)
		msg, err := reader.ReadBytes('}')

		// Parse serial data
		if err != nil {
			status.Log(logging.Error(), "Failed to read from serial port")
			status.Log(logging.Error(), err.Error())
			connected = false
			break
		} else {
			var data interface{}
			json.Unmarshal(msg, &data)
			if isWriter {
				go Pop(serialDevice)
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf("Disconnected from serial")
}

// Push queues a message for writing
func Push(device *serial.Port, msg string) {
	writeQueueLock.Lock()
	defer writeQueueLock.Unlock()
	_, ok := writeQueue[device]
	if !ok {
		writeQueue[device] = []string{}
	}

	if !formatting.StringInSlice(msg, writeQueue[device]) {
		writeQueue[device] = append(writeQueue[device], msg)
	}
}

// Pop the last message off the queue and write it to the respective serial
func Pop(device *serial.Port) {
	if device == nil {
		status.Log(logging.Error(), "Serial port is not set, nothing to write to.")
		return
	}

	writeQueueLock.Lock()
	defer writeQueueLock.Unlock()
	queue, ok := writeQueue[device]
	if !ok || len(queue) == 0 {
		return
	}

	var msg string
	msg, writeQueue[device] = writeQueue[device][len(writeQueue[device])-1], writeQueue[device][:len(writeQueue[device])-1]

	write(device, msg)
}

// write pushes out a message to the open serial port
func write(device *serial.Port, msg string) {
	if device == nil {
		status.Log(logging.Error(), "Serial port is not set, nothing to write to.")
		return
	}

	if len(msg) == 0 {
		status.Log(logging.Warning(), "Empty message, not writing to serial")
		return
	}

	n, err := device.Write([]byte(msg))
	if err != nil {
		status.Log(logging.Error(), "Failed to write to serial port")
		status.Log(logging.Error(), err.Error())
		return
	}

	status.Log(logging.OK(), fmt.Sprintf("Successfully wrote %s (%d bytes) to serial.", msg, n))
}
