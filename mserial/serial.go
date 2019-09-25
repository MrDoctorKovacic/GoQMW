package mserial

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
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

// ReadSerial will continuously pull data from incoming serial
func ReadSerial(serialDevice *serial.Port) interface{} {
	reader := bufio.NewReader(serialDevice)

	// While connected, try to read from the device
	// If we become disconnected, the goroutine will end and will have to be restarted
	for connected := true; connected; {
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

			return data
		}
	}
	return nil
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
