package mserial

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/tarm/serial"
)

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
				status.Log(logging.Error(), "Failed to convert given baudrate string to int. Found values: "+deviceName+": "+baudrateString)
				return nil
			}
			devices[deviceName] = deviceBaud
		}
	}

	return devices
}

// MachineShutdown shutdowns the named machine safely
func MachineShutdown(serialDevice *serial.Port, machine string, timeToSleep time.Duration, serialMessage string) {
	CommandNetworkMachine(machine, "shutdown")
	time.Sleep(timeToSleep)
	Push(serialDevice, serialMessage)
}

// CommandNetworkMachine sends a command to a network machine, using a simple python server to recieve
func CommandNetworkMachine(name string, command string) {
	machineServiceAddress, err := settings.Get(formatting.FormatName(name), "ADDRESS")
	if machineServiceAddress == "" {
		return
	}

	resp, err := http.Get(fmt.Sprintf("http://%s:5350/%s", machineServiceAddress, command))
	if err != nil {
		status.Log(logging.Error(), fmt.Sprintf("Failed to command machine %s (at %s) to %s: \n%s", name, machineServiceAddress, command, err.Error()))
		return
	}
	defer resp.Body.Close()
}
