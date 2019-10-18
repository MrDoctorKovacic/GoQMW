package sessions

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/rs/zerolog/log"
	"github.com/tarm/serial"
)

// StartSerialComms will set up the serial port,
// and start the ReadSerial goroutine
func StartSerialComms(deviceName string, baudrate int) {
	log.Info().Msg("Opening serial device " + deviceName)
	c := &serial.Config{Name: deviceName, Baud: baudrate, ReadTimeout: time.Second * 10}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Error().Msg("Failed to open serial port " + deviceName)
		log.Error().Msg(err.Error())
		return
	}

	var isWriter bool

	// Use first Serial device as a R/W, all others will only be read from
	if settings.SerialControlDevice == nil {
		settings.SerialControlDevice = s
		isWriter = true
		log.Info().Msg("Using serial device " + deviceName + " as default writer")
	}

	// Continiously read from serial port
	endedSerial := ReadFromSerial(s, isWriter)
	if endedSerial {
		log.Error().Msg("Serial disconnected, closing port and reopening")

		// Replace main serial writer
		if settings.SerialControlDevice == s {
			settings.SerialControlDevice = nil
		}

		s.Close()
		time.Sleep(time.Second * 10)
		log.Error().Msg("Reopening serial port...")
		StartSerialComms(deviceName, baudrate)
	}
}

// ReadFromSerial reads serial data into the session
func ReadFromSerial(device *serial.Port, isWriter bool) bool {
	log.Info().Msg("Starting serial read")
	for connected := true; connected; {
		response, err := mserial.ReadSerial(device, isWriter)
		if err != nil {
			// The device is nil, break out of this read loop
			log.Error().Msg(err.Error())
			break
		}
		parseSerialJSON(response)
	}
	return true
}

func parseSerialJSON(marshalledJSON interface{}) {
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
			log.Error().Msg(fmt.Sprintf("%s is of a type I don't know how to handle (%s: %s)", key, vv, value))
		}
	}
}
