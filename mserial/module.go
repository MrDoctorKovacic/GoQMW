package mserial

import (
	"bufio"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/format"
	"github.com/rs/zerolog/log"
	"github.com/tarm/serial"
)

// Module exports MDroid module
type Module struct{}

// Message for the serial writer, and a channel to await it
type Message struct {
	Device     *serial.Port
	Text       string
	isComplete chan error
	UUID       string
}

var (
	// Writer is our one main port to default to
	Writer *serial.Port
	// Mod exports our module functionality
	Mod            Module
	writeQueue     map[*serial.Port][]*Message
	writeQueueLock sync.Mutex
)

func init() {
	writeQueue = make(map[*serial.Port][]*Message, 0)
}

// Setup handles module init
func (*Module) Setup(configAddr *map[string]string) {
	configMap := *configAddr

	hardwareSerialPort, usingHardwareSerial := configMap["HARDWARE_SERIAL_PORT"]
	if !usingHardwareSerial {
		log.Warn().Msgf("No hardware serial port defined. Not setting up serial devices.")
		return
	}

	// Check if serial is required for startup
	// This allows setting an initial state without incorrectly triggering hooks
	serialRequiredSetting, ok := configMap["SERIAL_STARTUP"]
	if ok && serialRequiredSetting == "TRUE" {
		// Serial is required for setup.
		// Open a port, set state to the output and immediately close for later concurrent reading
		s, err := openSerialPort(hardwareSerialPort, 115200)
		if err != nil {
			log.Error().Msg(err.Error())
		}
		readSerial(s)
		log.Info().Msg("Closing port for later reading")
		s.Close()
	}

	// Start initial reader / writer
	log.Info().Msgf("Registering %s as serial writer", hardwareSerialPort)
	go startSerialComms(hardwareSerialPort, 115200)

	// Setup other devices
	for device, baudrate := range parseSerialDevices(settings.GetAll()) {
		go startSerialComms(device, baudrate)
	}
}

// SetRoutes inits module routes
func (*Module) SetRoutes(router *mux.Router) {
	//
	// Serial routes
	//
	router.HandleFunc("/serial/{command}/{checksum}", WriteSerialHandler).Methods("POST", "GET")
	router.HandleFunc("/serial/{command}", WriteSerialHandler).Methods("POST", "GET")
}

// WriteSerialHandler handles messages sent through the server
func WriteSerialHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	if params["command"] != "" {
		AwaitText(params["command"])
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
