package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/gorilla/mux"
	"github.com/tarm/serial"
)

// SerialStatus will control logging and reporting of status / warnings / errors
var SerialStatus = logging.NewStatus("Serial")

// Sub function to parse through other serial devices, if enabled
func parseSerialDevices(settingsData map[string]map[string]string) {

	serialDevices, additionalSerialDevices := settingsData["TLV"]

	if additionalSerialDevices {
		// Loop through each READONLY serial device and set up
		// No room to config baud rate here, use 9600 as default
		for deviceName, baudrateString := range serialDevices {
			deviceBaud, err := strconv.Atoi(baudrateString)
			if err != nil {
				MainStatus.Log(logging.Error(), "Failed to convert given baudrate string to int. Found values: "+deviceName+": "+baudrateString)
				return
			}

			StartSerialComms(deviceName, deviceBaud)
		}
	}
}

func parseSerialJSON(marshalledJSON interface{}) {

	if marshalledJSON == nil {
		SerialStatus.Log(logging.Error(), " marshalled JSON is nil.")
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
			SerialStatus.Log(logging.Error(), key+" is an array. Data: ")
			for i, u := range vv {
				fmt.Println(i, u)
			}
		default:
			SerialStatus.Log(logging.Error(), key+" is of a type I don't know how to handle")
		}
	}
}

// WriteSerialHandler handles messages sent through the server
func WriteSerialHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	if params["command"] != "" {
		WriteSerial(params["command"])
	}
	json.NewEncoder(w).Encode("OK")
}

// ReadSerial will continuously pull data from incoming serial
func ReadSerial(serialDevice *serial.Port) {
	serialReads := 0
	reader := bufio.NewReader(serialDevice)
	SerialStatus.Log(logging.OK(), "Starting serial read")

	// While connected, try to read from the device
	// If we become disconnected, the goroutine will end and will have to be restarted
	for connected := true; connected; serialReads++ {
		//buf := make([]byte, 1024)
		//n, err := serialDevice.Read(buf)
		msg, err := reader.ReadBytes('}')

		// Parse serial data
		if err != nil {
			SerialStatus.Log(logging.Error(), "Failed to read from serial port")
			SerialStatus.Log(logging.Error(), err.Error())
			connected = false
		} else {
			var data interface{}
			SerialStatus.Log(logging.OK(), string(msg))
			json.Unmarshal(msg, &data)

			parseSerialJSON(data)
		}
	}
}

// WriteSerial pushes out a message to the open serial port
func WriteSerial(msg string) {
	if Config.SerialControlDevice == nil {
		SerialStatus.Log(logging.Error(), "Serial port is not set, nothing to write to.")
		return
	}

	if len(msg) == 0 {
		SerialStatus.Log(logging.Warning(), "Empty message, not writing to serial")
		return
	}

	n, err := Config.SerialControlDevice.Write([]byte(msg))
	if err != nil {
		SerialStatus.Log(logging.Error(), "Failed to write to serial port")
		SerialStatus.Log(logging.Error(), err.Error())
		return
	}

	SerialStatus.Log(logging.OK(), fmt.Sprintf("Successfully wrote %s (%d bytes) to serial.", msg, n))
}

// StartSerialComms will set up the serial port,
// and start the ReadSerial goroutine
func StartSerialComms(deviceName string, baudrate int) {
	SerialStatus.Log(logging.OK(), "Opening serial device "+deviceName)
	c := &serial.Config{Name: deviceName, Baud: baudrate}
	s, err := serial.OpenPort(c)
	if err != nil {
		SerialStatus.Log(logging.Error(), "Failed to open serial port "+deviceName)
		SerialStatus.Log(logging.Error(), err.Error())
	} else {
		// Use first Serial device as a R/W, all others will only be read from
		if Config.SerialControlDevice == nil {
			Config.SerialControlDevice = s
			SerialStatus.Log(logging.OK(), "Using serial device "+deviceName+" as default writer")
		}

		// Continiously read from serial port
		go ReadSerial(s)
	}

}
