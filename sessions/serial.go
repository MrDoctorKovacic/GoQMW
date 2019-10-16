package sessions

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/tarm/serial"
)

// StartSerialComms will set up the serial port,
// and start the ReadSerial goroutine
func StartSerialComms(deviceName string, baudrate int) {
	status.Log(logging.OK(), "Opening serial device "+deviceName)
	c := &serial.Config{Name: deviceName, Baud: baudrate, ReadTimeout: time.Second * 10}
	s, err := serial.OpenPort(c)
	if err != nil {
		status.Log(logging.Error(), "Failed to open serial port "+deviceName)
		status.Log(logging.Error(), err.Error())
		return
	}

	var isWriter bool

	// Use first Serial device as a R/W, all others will only be read from
	if settings.Config.SerialControlDevice == nil {
		settings.Config.SerialControlDevice = s
		isWriter = true
		status.Log(logging.OK(), "Using serial device "+deviceName+" as default writer")
	}

	// Continiously read from serial port
	endedSerial := ReadFromSerial(s, isWriter)
	if endedSerial {
		status.Log(logging.Error(), "Serial disconnected, closing port and reopening")

		// Replace main serial writer
		if settings.Config.SerialControlDevice == s {
			settings.Config.SerialControlDevice = nil
		}

		s.Close()
		time.Sleep(time.Second * 10)
		status.Log(logging.Error(), "Reopening serial port...")
		StartSerialComms(deviceName, baudrate)
	}
}

// ReadFromSerial reads serial data into the session
func ReadFromSerial(device *serial.Port, isWriter bool) bool {
	status.Log(logging.OK(), "Starting serial read")
	for connected := true; connected; {
		response, err := mserial.ReadSerial(device, isWriter)
		if err != nil {
			// The device is nil, break out of this read loop
			status.Log(logging.Error(), err.Error())
			break
		}
		parseSerialJSON(response)
	}
	return true
}

func parseSerialJSON(marshalledJSON interface{}) {
	if marshalledJSON == nil {
		status.Log(logging.Error(), " marshalled JSON is nil.")
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
			floatValue, ok := value.(float32)
			if ok {
				SetValue(strings.ToUpper(key), fmt.Sprintf("%f", floatValue))
			}
		case float64:
			floatValue, ok := value.(float64)
			if ok {
				SetValue(strings.ToUpper(key), fmt.Sprintf("%f", floatValue))
			}
		case []interface{}:
			status.Log(logging.Error(), key+" is an array. Data: ")
			for i, u := range vv {
				fmt.Println(i, u)
			}
		case nil:
			break
		default:
			status.Log(logging.Error(), fmt.Sprintf("%s is of a type I don't know how to handle (%s: %s)", key, vv, value))
		}
	}
}
