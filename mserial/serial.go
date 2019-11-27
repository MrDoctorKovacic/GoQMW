package mserial

import (
	"bufio"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"github.com/MrDoctorKovacic/MDroid-Core/format"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/tarm/serial"
)

var (
	// Writer is our one main port to default to
	Writer         *serial.Port
	writeQueue     map[*serial.Port][]string
	writeQueueLock sync.Mutex
)

func init() {
	writeQueue = make(map[*serial.Port][]string, 0)
}

// ParseSerialDevices parses through other serial devices, if enabled
func ParseSerialDevices(settingsData map[string]map[string]string) map[string]int {

	serialDevices, additionalSerialDevices := settingsData["Serial Devices"]
	var devices map[string]int

	if additionalSerialDevices {
		for deviceName, baudrateString := range serialDevices {
			deviceBaud, err := strconv.Atoi(baudrateString)
			if err != nil {
				log.Error().Msgf("Failed to convert given baudrate string to int. Found values: %s: %s", deviceName, baudrateString)
			} else {
				devices[deviceName] = deviceBaud
			}
		}
	}

	return devices
}

// WriteSerialHandler handles messages sent through the server
func WriteSerialHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	if params["command"] != "" {
		Push(Writer, params["command"])
	}
	format.WriteResponse(&w, r, format.JSONResponse{Output: "OK", OK: true})
}

// Read will continuously pull data from incoming serial
func Read(serialDevice *serial.Port) (interface{}, error) {
	reader := bufio.NewReader(serialDevice)
	msg, err := reader.ReadBytes('}')
	if err != nil {
		return nil, err
	}

	// Parse serial data
	var data interface{}
	json.Unmarshal(msg, &data)
	return data, nil
}

// Push queues a message for writing
func Push(device *serial.Port, msg string) {
	writeQueueLock.Lock()
	defer writeQueueLock.Unlock()
	_, ok := writeQueue[device]
	if !ok {
		writeQueue[device] = []string{}
	}

	writeQueue[device] = append(writeQueue[device], msg)
}

// Pop the last message off the queue and write it to the respective serial
func Pop(device *serial.Port) {
	if device == nil {
		log.Error().Msg("Serial port is not set, nothing to write to.")
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
		log.Error().Msg("Serial port is not set, nothing to write to.")
		return
	}

	if len(msg) == 0 {
		log.Warn().Msg("Empty message, not writing to serial")
		return
	}

	n, err := device.Write([]byte(msg))
	if err != nil {
		log.Error().Msg("Failed to write to serial port")
		log.Error().Msg(err.Error())
		return
	}

	log.Info().Msgf("Successfully wrote %s (%d bytes) to serial.", msg, n)
}
