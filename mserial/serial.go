package mserial

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/qcasey/MDroid-Core/sessions"
	"github.com/rs/zerolog/log"
	"github.com/tarm/serial"
)

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

// openSerialPort will return a *serial.Port with the given arguments
func openSerialPort(deviceName string, baudrate int) (*serial.Port, error) {
	log.Info().Msgf("Opening serial device %s at baud %d", deviceName, baudrate)
	c := &serial.Config{Name: deviceName, Baud: baudrate, ReadTimeout: time.Second * 10}
	s, err := serial.OpenPort(c)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// startSerialComms will set up the serial port,
// and start the ReadSerial goroutine
func startSerialComms(deviceName string, baudrate int) {
	s, err := openSerialPort(deviceName, baudrate)
	if err != nil {
		log.Error().Msgf("Failed to open serial port %s", deviceName)
		log.Error().Msg(err.Error())
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
	loop(s, isWriter) // this will block until abrubtly ended
	log.Error().Msg("Serial disconnected, closing port and reopening in 10 seconds")

	// Replace main serial writer
	if Writer == s {
		Writer = nil
	}

	s.Close()
	time.Sleep(time.Second * 10)
	log.Error().Msg("Reopening serial port...")
	go startSerialComms(deviceName, baudrate)
}

// loop reads serial data into the session
func loop(device *serial.Port, isWriter bool) {
	for {
		// Write to device if is necessary
		if isWriter {
			Pop(device)
		}

		err := readSerial(device)
		if err != nil {
			// The device is nil, break out of this read loop
			log.Error().Msg("Failed to read from serial port")
			log.Error().Msg(err.Error())
			return
		}
	}
}

// readSerial takes one line from the serial device and parses it into the session
func readSerial(device *serial.Port) error {
	response, err := read(device)
	if err != nil {
		return err
	}

	// Parse serial data
	parseJSON(response)
	return nil
}

// read will continuously pull data from incoming serial
func read(serialDevice *serial.Port) (interface{}, error) {
	reader := bufio.NewReader(serialDevice)
	msg, _, err := reader.ReadLine()
	if err != nil {
		return nil, err
	}

	// Parse serial data
	var data interface{}
	json.Unmarshal(msg, &data)
	return data, nil
}

// write pushes out a message to the open serial port
func write(msg *Message) error {
	if msg.Device == nil {
		return fmt.Errorf("Serial port is not set, nothing to write to")
	}

	if len(msg.Text) == 0 {
		return fmt.Errorf("Empty message, not writing to serial")
	}

	n, err := msg.Device.Write([]byte(msg.Text))
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

func parseJSON(marshalledJSON interface{}) {
	if marshalledJSON == nil {
		log.Debug().Msg("Marshalled JSON is nil.")
		return
	}

	data := marshalledJSON.(map[string]interface{})

	// Switch through various types of JSON data
	for key, value := range data {
		switch vv := value.(type) {
		case bool:
			sessions.SetValue(strings.ToUpper(key), strings.ToUpper(strconv.FormatBool(vv)))
		case string:
			sessions.SetValue(strings.ToUpper(key), strings.ToUpper(vv))
		case int:
			sessions.SetValue(strings.ToUpper(key), strconv.Itoa(value.(int)))
		case float32:
			if floatValue, ok := value.(float32); ok {
				sessions.SetValue(strings.ToUpper(key), fmt.Sprintf("%f", floatValue))
			}
		case float64:
			if floatValue, ok := value.(float64); ok {
				sessions.SetValue(strings.ToUpper(key), fmt.Sprintf("%f", floatValue))
			}
		case []interface{}:
			log.Error().Msg(key + " is an array. Data: ")
			for i, u := range vv {
				fmt.Println(i, u)
			}
		case nil:
			break
		default:
			log.Error().Msgf("%s is of a type I don't know how to handle (%s: %s)", key, vv, value)
		}
	}
}
