package mserial

import (
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/format"
	"github.com/qcasey/MDroid-Core/format/response"
	"github.com/qcasey/MDroid-Core/settings"
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

// Setup handles module init
func Setup(router *mux.Router) {
	if !settings.Data.IsSet("mdroid.HARDWARE_SERIAL_PORT") {
		log.Warn().Msgf("No hardware serial port defined. Not setting up serial devices.")
		return
	}

	hardwareSerialPort := settings.Data.GetString("mdroid.HARDWARE_SERIAL_PORT")

	// Start initial reader / writer
	log.Info().Msgf("Registering %s as serial writer", hardwareSerialPort)
	go startSerialComms(hardwareSerialPort, 115200)

	// Setup other devices
	/*
		for device, baudrate := range parseSerialDevices(settings.GetAll()) {
			go startSerialComms(device, baudrate)
		}*/

	//
	// Serial routes
	//
	router.HandleFunc("/serial/{command}", WriteSerialHandler).Methods("POST", "GET")
	router.HandleFunc("/gyros", getGyroMeasurements).Methods("GET")
}

// WriteSerialHandler handles messages sent through the server
func WriteSerialHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	if params["command"] != "" {
		AwaitText(params["command"])
	}
	response.WriteNew(&w, r, response.JSONResponse{Output: "OK", OK: true})
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
	m.UUID, _ = format.NewShortUUID()
	m.isComplete = make(chan error)
	log.Info().Msgf("[%s] Awaiting serial message write", m.UUID)
	Push(m)
	err := <-m.isComplete
	log.Info().Msgf("[%s] Message write is complete", m.UUID)
	return err
}

// AwaitText creates a new message with the default writer, appends it for sending, and waits for it to be sent
func AwaitText(message string) error {
	uuid, _ := format.NewShortUUID()
	m := &Message{Device: Writer, Text: message, isComplete: make(chan error), UUID: uuid}
	log.Info().Msgf("[%s] Awaiting serial message write", m.UUID)
	Push(m)
	err := <-m.isComplete
	log.Info().Msgf("[%s] Message write is complete", m.UUID)
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

	err := write(msg)
	msg.isComplete <- err
}
