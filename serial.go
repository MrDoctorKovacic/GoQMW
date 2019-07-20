package main

import (
	"encoding/json"
	"fmt"

	"github.com/MrDoctorKovacic/MDroid-Core/status"
	"github.com/tarm/serial"
)

// HardwareReadout holds the name and power state of various hardware
type HardwareReadout struct {
	TABLET_POWER string `json:"TABLET_POWER,omitempty"`
	BOARD_POWER  string `json:"BOARD_POWER,omitempty"`
	ACC_POWER    string `json:"ACC_POWER,omitempty"`
}

// SerialStatus will control logging and reporting of status / warnings / errors
var SerialStatus = status.NewStatus("Serial")

// serialDevice is the
var serialDevice *serial.Port

// ReadSerial will continuously pull data from incoming serial
func ReadSerial() {
	serialReads := 0

	// While connected, try to read from the device
	// If we become disconnected, the goroutine will end and will have to be restarted
	for connected := true; connected; serialReads++ {
		buf := make([]byte, 128)
		n, err := serialDevice.Read(buf)

		// Parse serial data
		if err != nil {
			SerialStatus.Log(status.Error(), "Failed to read from serial port")
			SerialStatus.Log(status.Error(), err.Error())
			connected = false
		} else {
			var data HardwareReadout
			json.Unmarshal(buf[:n], &data)

			if data.TABLET_POWER != "" {
				SetSessionRawValue("TABLET_POWER", data.TABLET_POWER)
			}

			if data.BOARD_POWER != "" {
				SetSessionRawValue("BOARD_POWER", data.BOARD_POWER)
			}

			if data.ACC_POWER != "" {
				SetSessionRawValue("ACC_POWER", data.ACC_POWER)
			}
		}
	}
}

func WriteSerial(msg string) {
	if len(msg) == 0 {
		SerialStatus.Log(status.Warning(), "Empty message, not writing to serial")
		return
	}

	if serialDevice == nil {
		SerialStatus.Log(status.Error(), "Serial port is not set, nothing to write to.")
		return
	}

	n, err := serialDevice.Write([]byte(msg))
	if err != nil {
		SerialStatus.Log(status.Error(), "Failed to write to serial port")
		SerialStatus.Log(status.Error(), err.Error())
		return
	}

	SerialStatus.Log(status.OK(), fmt.Sprintf("Successfully wrote %s (%d bytes) to serial.", msg, n))
}

func StartSerialComms(deviceName string, baudrate int) {
	c := &serial.Config{Name: deviceName, Baud: baudrate}
	s, err := serial.OpenPort(c)
	if err != nil {
		SerialStatus.Log(status.Error(), "Failed to open serial port "+deviceName)
		SerialStatus.Log(status.Error(), err.Error())
	} else {
		serialDevice = s
		// Continiously read from serial port
		go ReadSerial()
	}

}
