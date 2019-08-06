package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/MrDoctorKovacic/MDroid-Core/status"
	"github.com/tarm/serial"
)

// SerialStatus will control logging and reporting of status / warnings / errors
var SerialStatus = status.NewStatus("Serial")

func parseSerialJSON(marshalledJSON interface{}) {

	if marshalledJSON == nil {
		SerialStatus.Log(status.Error(), " marshalled JSON is nil.")
		return
	}

	data := marshalledJSON.(map[string]interface{})

	// Switch through various types of JSON data
	for key, value := range data {
		switch vv := value.(type) {
		case bool:
			SetSessionRawValue(strings.ToUpper(key), strings.ToUpper(strconv.FormatBool(vv)))
		case string:
			SetSessionRawValue(strings.ToUpper(key), strings.ToUpper(vv))
		case int:
			SetSessionRawValue(strings.ToUpper(key), strconv.Itoa(vv))
		case []interface{}:
			SerialStatus.Log(status.Error(), key+" is an array. Data: ")
			for i, u := range vv {
				fmt.Println(i, u)
			}
		default:
			SerialStatus.Log(status.Error(), key+" is of a type I don't know how to handle")
		}
	}
}

// ReadSerial will continuously pull data from incoming serial
func ReadSerial(serialDevice *serial.Port) {
	serialReads := 0
	reader := bufio.NewReader(serialDevice)
	SerialStatus.Log(status.OK(), "Starting serial read")

	// While connected, try to read from the device
	// If we become disconnected, the goroutine will end and will have to be restarted
	for connected := true; connected; serialReads++ {
		//buf := make([]byte, 1024)
		//n, err := serialDevice.Read(buf)
		msg, err := reader.ReadBytes('}')

		// Parse serial data
		if err != nil {
			SerialStatus.Log(status.Error(), "Failed to read from serial port")
			SerialStatus.Log(status.Error(), err.Error())
			connected = false
		} else {
			var data interface{}
			json.Unmarshal(msg, &data)

			parseSerialJSON(data)
		}
	}
}

// WriteSerial pushes out a message to the open serial port
func WriteSerial(msg string) {
	if len(msg) == 0 {
		SerialStatus.Log(status.Warning(), "Empty message, not writing to serial")
		return
	}

	if Config.SerialControlDevice == nil {
		SerialStatus.Log(status.Error(), "Serial port is not set, nothing to write to.")
		return
	}

	n, err := Config.SerialControlDevice.Write([]byte(msg))
	if err != nil {
		SerialStatus.Log(status.Error(), "Failed to write to serial port")
		SerialStatus.Log(status.Error(), err.Error())
		return
	}

	SerialStatus.Log(status.OK(), fmt.Sprintf("Successfully wrote %s (%d bytes) to serial.", msg, n))
}

// StartSerialComms will set up the serial port,
// and start the ReadSerial goroutine
func StartSerialComms(deviceName string, baudrate int) {
	SerialStatus.Log(status.OK(), "Opening serial device "+deviceName)
	c := &serial.Config{Name: deviceName, Baud: baudrate}
	s, err := serial.OpenPort(c)
	if err != nil {
		SerialStatus.Log(status.Error(), "Failed to open serial port "+deviceName)
		SerialStatus.Log(status.Error(), err.Error())
	} else {
		// Use first Serial device as a R/W, all others will only be read from
		if Config.SerialControlDevice == nil {
			Config.SerialControlDevice = s
			SerialStatus.Log(status.OK(), "Using serial device "+deviceName+" as default writer")
		}

		// Continiously read from serial port
		go ReadSerial(s)
	}

}
