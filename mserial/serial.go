package mserial

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/tarm/serial"
)

// SerialStatus will control logging and reporting of status / warnings / errors
var SerialStatus = logging.NewStatus("Serial")

// ParseSerialDevices parses through other serial devices, if enabled
func ParseSerialDevices(settingsData map[string]map[string]string) map[string]int {

	serialDevices, additionalSerialDevices := settingsData["TLV"]
	var devices map[string]int

	if additionalSerialDevices {
		// Loop through each READONLY serial device and set up
		// No room to config baud rate here, use 9600 as default
		for deviceName, baudrateString := range serialDevices {
			deviceBaud, err := strconv.Atoi(baudrateString)
			if err != nil {
				SerialStatus.Log(logging.Error(), "Failed to convert given baudrate string to int. Found values: "+deviceName+": "+baudrateString)
				return nil
			}
			devices[deviceName] = deviceBaud
		}
	}

	return devices
}

func parseSerialJSON(marshalledJSON interface{}, session *sessions.Session) {

	if marshalledJSON == nil {
		SerialStatus.Log(logging.Error(), " marshalled JSON is nil.")
		return
	}

	data := marshalledJSON.(map[string]interface{})

	// Switch through various types of JSON data
	for key, value := range data {
		switch vv := value.(type) {
		case bool:
			session.CreateSessionValue(strings.ToUpper(key), strings.ToUpper(strconv.FormatBool(vv)))
		case string:
			session.CreateSessionValue(strings.ToUpper(key), strings.ToUpper(vv))
		case int:
			session.CreateSessionValue(strings.ToUpper(key), strconv.Itoa(value.(int)))
		case float32:
			floatValue, _ := value.(float32)
			session.CreateSessionValue(strings.ToUpper(key), fmt.Sprintf("%f", floatValue))
		case float64:
			floatValue, _ := value.(float64)
			session.CreateSessionValue(strings.ToUpper(key), fmt.Sprintf("%f", floatValue))
		case []interface{}:
			SerialStatus.Log(logging.Error(), key+" is an array. Data: ")
			for i, u := range vv {
				fmt.Println(i, u)
			}
		case nil:
			break
		default:
			SerialStatus.Log(logging.Error(), fmt.Sprintf("%s is of a type I don't know how to handle (%s: %s)", key, vv, value))
		}
	}
}

// ReadSerial will continuously pull data from incoming serial
func ReadSerial(serialDevice *serial.Port, session *sessions.Session) {
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
			json.Unmarshal(msg, &data)

			parseSerialJSON(data, session)
		}
	}
}

// WriteSerial pushes out a message to the open serial port
func WriteSerial(device *serial.Port, msg string) {
	if device == nil {
		SerialStatus.Log(logging.Error(), "Serial port is not set, nothing to write to.")
		return
	}

	if len(msg) == 0 {
		SerialStatus.Log(logging.Warning(), "Empty message, not writing to serial")
		return
	}

	n, err := device.Write([]byte(msg))
	if err != nil {
		SerialStatus.Log(logging.Error(), "Failed to write to serial port")
		SerialStatus.Log(logging.Error(), err.Error())
		return
	}

	SerialStatus.Log(logging.OK(), fmt.Sprintf("Successfully wrote %s (%d bytes) to serial.", msg, n))
}

// MachineShutdown shutdowns the named machine safely
func MachineShutdown(serialDevice *serial.Port, machine string, timeToSleep time.Duration, serialMessage string) {
	CommandNetworkMachine(machine, "shutdown")
	time.Sleep(timeToSleep)
	WriteSerial(serialDevice, serialMessage)
}

// CommandNetworkMachine sends a command to a network machine, using a simple python server to recieve
func CommandNetworkMachine(name string, command string) {
	machineServiceAddress, err := settings.Get(formatting.FormatName(name), "ADDRESS")
	if machineServiceAddress == "" {
		return
	}

	resp, err := http.Get(fmt.Sprintf("http://%s:5350/%s", machineServiceAddress, command))
	if err != nil {
		SerialStatus.Log(logging.Error(), fmt.Sprintf("Failed to command machine %s (at %s) to %s: \n%s", name, machineServiceAddress, command, err.Error()))
		return
	}
	defer resp.Body.Close()
}
