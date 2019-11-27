package mserial

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/MrDoctorKovacic/MDroid-Core/format"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/tarm/serial"
)

// Message for the serial writer, and a channel to await it
type Message struct {
	Device     *serial.Port
	Text       string
	isComplete chan error
	UUID       string
}

var (
	// Writer is our one main port to default to
	Writer         *serial.Port
	writeQueue     map[*serial.Port][]*Message
	writeQueueLock sync.Mutex
)

func init() {
	writeQueue = make(map[*serial.Port][]*Message, 0)
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
		PushText(params["command"])
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
func Push(m *Message) {
	writeQueueLock.Lock()
	defer writeQueueLock.Unlock()
	_, ok := writeQueue[m.Device]
	if !ok {
		writeQueue[m.Device] = []*Message{}
	}

	writeQueue[m.Device] = append(writeQueue[m.Device], m)
}

// PushText creates a new message with the default writer, and appends it for sending
func PushText(message string) {
	m := &Message{Device: Writer, Text: message}
	Push(m)
}

// Await queues a message for writing, and waits for it to be sent
func Await(m *Message) error {
	m.UUID, _ = format.NewUUID()
	m.isComplete = make(chan error)
	log.Info().Msgf("Awaiting serial message write %s", m.UUID)
	Push(m)
	err := <-m.isComplete
	log.Info().Msgf("Message write %s is complete", m.UUID)
	return err
}

// AwaitText creates a new message with the default writer, appends it for sending, and waits for it to be sent
func AwaitText(message string) error {
	uuid, _ := format.NewUUID()
	m := &Message{Device: Writer, Text: message, isComplete: make(chan error), UUID: uuid}
	log.Info().Msgf("Awaiting serial message write %s", m.UUID)
	Push(m)
	err := <-m.isComplete
	log.Info().Msgf("Message write %s is complete", m.UUID)
	return err
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

	var msg *Message
	msg, writeQueue[device] = writeQueue[device][len(writeQueue[device])-1], writeQueue[device][:len(writeQueue[device])-1]

	err := write(msg.Device, msg.Text)
	msg.isComplete <- err
}

// write pushes out a message to the open serial port
func write(device *serial.Port, msg string) error {
	if device == nil {
		return fmt.Errorf("Serial port is not set, nothing to write to")
	}

	if len(msg) == 0 {
		return fmt.Errorf("Empty message, not writing to serial")
	}

	n, err := device.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("Failed to write to serial port: %s", err.Error())
	}

	log.Info().Msgf("Successfully wrote %s (%d bytes) to serial.", msg, n)
	return nil
}
