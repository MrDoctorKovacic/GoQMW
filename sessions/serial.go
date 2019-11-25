package sessions

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
	"github.com/rs/zerolog/log"
	"github.com/tarm/serial"
)

// OpenSerialPort will return a *serial.Port with the given arguments
func OpenSerialPort(deviceName string, baudrate int) (*serial.Port, error) {
	log.Info().Msgf("Opening serial device %s at baud %d", deviceName, baudrate)
	c := &serial.Config{Name: deviceName, Baud: baudrate, ReadTimeout: time.Second * 10}
	s, err := serial.OpenPort(c)
	defer s.Close()
	if err != nil {
		return nil, err
	}
	return s, nil
}

// StartSerialComms will set up the serial port,
// and start the ReadSerial goroutine
func StartSerialComms(deviceName string, baudrate int) {
	s, err := OpenSerialPort(deviceName, baudrate)
	if err != nil {
		log.Error().Msgf("Failed to open serial port %s", deviceName)
		log.Error().Msg(err.Error())
		return
	}
	defer s.Close()

	// Use first Serial device as a R/W, all others will only be read from
	isWriter := false
	if mserial.Writer == nil {
		mserial.Writer = s
		isWriter = true
		log.Info().Msgf("Using serial device %s as default writer", deviceName)
	}

	// Continually read from serial port
	log.Info().Msgf("Starting new serial reader on device %s", deviceName)
	SerialLoop(s, isWriter) // this will block until abrubtly ended
	log.Error().Msg("Serial disconnected, closing port and reopening in 10 seconds")

	// Replace main serial writer
	if mserial.Writer == s {
		mserial.Writer = nil
	}

	s.Close()
	time.Sleep(time.Second * 10)
	log.Error().Msg("Reopening serial port...")
	go StartSerialComms(deviceName, baudrate)
}

// SerialLoop reads serial data into the session
func SerialLoop(device *serial.Port, isWriter bool) {
	for {
		// Write to device if is necessary
		if isWriter {
			mserial.Pop(device)
		}

		err := ReadSerial(device)
		if err != nil {
			// The device is nil, break out of this read loop
			log.Error().Msg("Failed to read from serial port")
			log.Error().Msg(err.Error())
			return
		}
	}
}

// ReadSerial takes one line from the serial device and parses it into the session
func ReadSerial(device *serial.Port) error {
	response, err := mserial.Read(device)
	if err != nil {
		return err
	}

	// Parse serial data
	parseJSON(response)
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
			SetValue(strings.ToUpper(key), strings.ToUpper(strconv.FormatBool(vv)))
		case string:
			SetValue(strings.ToUpper(key), strings.ToUpper(vv))
		case int:
			SetValue(strings.ToUpper(key), strconv.Itoa(value.(int)))
		case float32:
			if floatValue, ok := value.(float32); ok {
				SetValue(strings.ToUpper(key), fmt.Sprintf("%f", floatValue))
			}
		case float64:
			if floatValue, ok := value.(float64); ok {
				SetValue(strings.ToUpper(key), fmt.Sprintf("%f", floatValue))
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
