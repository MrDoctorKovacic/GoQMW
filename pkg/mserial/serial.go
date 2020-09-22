package mserial

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/sessions"
	"github.com/mitchellh/mapstructure"
	"github.com/qcasey/MDroid-Core/internal/core"
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

// Measurement contains a simple X,Y,Z output from the IMU
type Measurement struct {
	X float64 `json:"X"`
	Y float64 `json:"Y"`
	Z float64 `json:"Z"`
}

var writerLock sync.Mutex

var (
	// Writer is our one main port to default to
	Writer         *serial.Port
	writeQueue     map[*serial.Port][]*Message
	writeQueueLock sync.Mutex
)

func init() {
	writeQueue = make(map[*serial.Port][]*Message, 0)
}

// Start will set up the serial port and ReadSerial goroutine
func Start(c *core.Core) {
	if !c.Settings.IsSet("mdroid.HARDWARE_SERIAL_PORT") {
		log.Warn().Msgf("No hardware serial port defined. Not setting up serial devices.")
		return
	}

	hardwareSerialPort := c.Settings.GetString("mdroid.HARDWARE_SERIAL_PORT")

	// Start initial reader / writer
	log.Info().Msgf("Registering %s as serial writer", hardwareSerialPort)
	go begin(c, hardwareSerialPort, 115200)

	// Setup other devices
	/*
		for device, baudrate := range parseSerialDevices(settings.GetAll()) {
			go startSerialComms(device, baudrate)
		}*/

}

func begin(c *core.Core, deviceName string, baudrate int) {
	log.Info().Msgf("Opening serial device %s at baud %d", deviceName, baudrate)
	serialConfig := &serial.Config{Name: deviceName, Baud: baudrate, ReadTimeout: time.Second * 10}
	s, err := serial.OpenPort(serialConfig)
	if err != nil {
		log.Error().Msgf("Failed to open serial port %s", deviceName)
		log.Error().Msg(err.Error())
		time.Sleep(time.Second * 2)
		go begin(c, deviceName, baudrate)
		return
	}
	defer s.Close()

	// Use first Serial device as a R/W, all others will only be read from
	isWriter := false
	if Writer == nil {
		Writer = s
		isWriter = true
		log.Info().Msgf("Using serial device %s as default writer", deviceName)
	}

	// Continually read from serial port
	log.Info().Msgf("Starting new serial reader on device %s", deviceName)
	for {
		// Write to device if is necessary
		if isWriter {
			Pop(s)
		}

		err := read(s)
		if err != nil {
			// The device is nil, break out of this read loop
			log.Error().Msg("Failed to read from serial port")
			log.Error().Msg(err.Error())
			break
		}
	}
	log.Error().Msg("Serial disconnected, closing port and reopening in 10 seconds")

	// Replace main serial writer
	if Writer == s {
		Writer = nil
	}

	s.Close()
	time.Sleep(time.Second * 10)
	log.Error().Msg("Reopening serial port...")
	go begin(c, deviceName, baudrate)
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
	//m.UUID, _ = format.NewShortUUID()
	m.isComplete = make(chan error)
	//log.Info().Msgf("[%s] Awaiting serial message write", m.UUID)
	Push(m)
	err := <-m.isComplete
	//log.Info().Msgf("[%s] Message write is complete", m.UUID)
	return err
}

// AwaitText creates a new message with the default writer, appends it for sending, and waits for it to be sent
func AwaitText(message string) error {
	//uuid, _ := format.NewShortUUID()
	m := &Message{Device: Writer, Text: message, isComplete: make(chan error)}
	//log.Info().Msgf("[%s] Awaiting serial message write", m.UUID)
	Push(m)
	err := <-m.isComplete
	//log.Info().Msgf("[%s] Message write is complete", m.UUID)
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

// parseSerialDevices parses through other serial devices, if enabled
/*
func parseSerialDevices(settingsData map[string]map[string]string) map[string]int {

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
}*/

// readSerial takes one line from the serial device and parses it into the session
func read(device *serial.Port) error {
	reader := bufio.NewReader(device)
	msg, _, err := reader.ReadLine()
	if err != nil {
		return err
	}

	if len(msg) == 0 {
		return nil
	}

	// Parse serial data
	var jsonData interface{}
	err = json.Unmarshal(msg, &jsonData)
	if err != nil {
		return err
	}
	if jsonData == nil {
		return nil
	}

	// Handle parse errors here instead of passing up
	data := jsonData.(map[string]interface{})
	// Switch through various types of JSON data
	for key, value := range data {
		switch vv := value.(type) {
		case bool:
			sessions.Set(key, vv, true)
		case int:
			sessions.Set(key, vv, true)
		case float64:
			sessions.Set(key, vv, true)
		case string:
			sessions.Set(key, vv, true)
		case map[string]interface{}:
			var m Measurement
			err := mapstructure.Decode(value, &m)
			if err != nil {
				return err
			}
			if key != "ACCELERATION" && key != "GYROSCOPE" && key != "MAGNETIC" {
				return fmt.Errorf("Measurement key %s not registered for input", key)
			}

			// Skip publishing values
			sessions.Set(fmt.Sprintf("gyros.%s.x", strings.ToLower(key)), m.X, false)
			sessions.Set(fmt.Sprintf("gyros.%s.y", strings.ToLower(key)), m.Y, false)
			sessions.Set(fmt.Sprintf("gyros.%s.z", strings.ToLower(key)), m.Z, false)
		case []interface{}:
			log.Error().Msg(key + " is an array. Data: ")
			for i, u := range vv {
				fmt.Println(i, u)
			}
		case nil:
			break
		default:
			return fmt.Errorf("%s is of a type I don't know how to handle (%s: %s)", key, vv, value)
		}
	}

	return nil
}

// write pushes out a message to the open serial port
func write(msg *Message) error {
	if msg.Device == nil {
		return fmt.Errorf("Serial port is not set, nothing to write to")
	}

	if len(msg.Text) == 0 {
		return fmt.Errorf("Empty message, not writing to serial")
	}

	writerLock.Lock()
	n, err := msg.Device.Write([]byte(msg.Text))
	writerLock.Unlock()
	if err != nil {
		return fmt.Errorf("Failed to write to serial port: %s", err.Error())
	}

	if msg.UUID == "" {
		log.Info().Msgf("Successfully wrote %s (%d bytes) to serial.", msg.Text, n)
	} else {
		log.Info().Msgf("[%s] Successfully wrote %s (%d bytes) to serial.", msg.UUID, msg.Text, n)
	}
	return nil
}
