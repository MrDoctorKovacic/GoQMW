package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"strconv"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/bluetooth"
	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/influx"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/gorilla/mux"
	"github.com/tarm/serial"
)

// Config defined here, to be saved to below
var Config settings.ConfigValues

// MainSession object
var MainSession sessions.Session

// Main config parsing
func parseConfig() {

	flag.StringVar(&Config.SettingsFile, "settings-file", "", "File to recover the persistent settings.")
	flag.StringVar(&Config.SessionFile, "session-file", "", "[DEBUG ONLY] File to save and recover the last-known session.")
	flag.Parse()

	// Parse settings file
	settingsData, VerboseOutput := settings.ReadFile(Config.SettingsFile)
	Config.VerboseOutput = VerboseOutput

	// Check settings
	if _, err := json.Marshal(settingsData); err != nil {
		panic(err)
	}

	// Init session tracking (with or without Influx)
	// Fetch and append old session from disk if allowed
	MainSession = sessions.CreateSession(Config.SettingsFile)
	MainSession.Config = &Config

	// Parse through config if found in settings file
	configMap, ok := settingsData["MDROID"]
	if ok {

		setupTimezone(&configMap)
		setupDatabase(&configMap)
		setupBluetooth(&configMap)

		// Set up pybus repeat commands
		_, usingPybus := configMap["PYBUS_DEVICE"]
		if usingPybus {
			go MainSession.RepeatCommand("requestIgnitionStatus", 10)
			go MainSession.RepeatCommand("requestLampStatus", 20)
			go MainSession.RepeatCommand("requestVehicleStatus", 30)
			go MainSession.RepeatCommand("requestOdometer", 45)
			go MainSession.RepeatCommand("requestTimeStatus", 60)
			go MainSession.RepeatCommand("requestTemperatureStatus", 120)
		}

		// Debug session log
		Config.DebugSessionFile = configMap["DEBUG_SESSION_LOG"]

		// Slack URL
		Config.SlackURL = configMap["SLACK_URL"]

		// Set up Auth tokens
		authToken, usingAuth := configMap["AUTH_TOKEN"]
		if usingAuth {
			Config.AuthToken = authToken
		}

		setupSerial(&settingsData)

	} else {
		MainStatus.Log(logging.Warning(), "No config found in settings file, not parsing through config")
	}
}

func setupTimezone(configAddr *map[string]string) {
	configMap := *configAddr
	timezoneLocation, usingTimezone := configMap["Timezone"]
	if usingTimezone {
		loc, err := time.LoadLocation(timezoneLocation)
		if err == nil {
			Config.Timezone = loc
		} else {
			// If timezone has errored
			Config.Timezone, _ = time.LoadLocation("UTC")
		}
	} else {
		// If timezone is not set in config
		Config.Timezone, _ = time.LoadLocation("UTC")
	}
}

// Set up InfluxDB time series logging
func setupDatabase(configAddr *map[string]string) {
	configMap := *configAddr
	databaseHost, usingDatabase := configMap["DATABASE_HOST"]
	if usingDatabase {
		Config.DB = &influx.Influx{Host: databaseHost, Database: configMap["DATABASE_NAME"]}
		Config.DatabaseEnabled = true

		// Set up ping functionality
		// Proprietary pinging for component tracking
		if configMap["PING_HOST"] != "" {
			logging.RemotePingAddress = configMap["PING_HOST"]
		} else {
			MainStatus.Log(logging.OK(), "Not forwarding pings to host")
		}

	} else {
		Config.DatabaseEnabled = false
		MainStatus.Log(logging.OK(), "Not logging to influx db")
	}
}

func setupBluetooth(configAddr *map[string]string) {
	configMap := *configAddr
	bluetoothAddress, usingBluetooth := configMap["BLUETOOTH_ADDRESS"]
	if usingBluetooth {
		bluetooth.EnableAutoRefresh()
		bluetooth.SetAddress(bluetoothAddress)
		Config.BluetoothAddress = bluetoothAddress
	}
	Config.BluetoothEnabled = usingBluetooth
}

//
// PROPRIETARY
// Configure hardware serials, should not be used outside my own config
//

// WriteSerialHandler handles messages sent through the server
func WriteSerialHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	if params["command"] != "" {
		mserial.WriteSerial(Config.SerialControlDevice, params["command"])
	}
	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "OK", Status: "success", OK: true})
}

func setupSerial(configAddr *map[string]map[string]string) {
	settingsData := *configAddr
	configMap := settingsData["MDROID"]
	HardwareSerialPort, usingHardwareSerial := configMap["HARDWARE_SERIAL_PORT"]
	hardwareSerialBaud, usingHardwareBaud := configMap["HARDWARE_SERIAL_BAUD"]
	Config.HardwareSerialEnabled = usingHardwareSerial

	if Config.HardwareSerialEnabled {
		// Configure default baudrate
		HardwareSerialBaud := 9600
		if usingHardwareBaud {
			baudrateString, err := strconv.Atoi(hardwareSerialBaud)
			if err != nil {
				MainStatus.Log(logging.Error(), "Failed to convert HardwareSerialBaud to int. Found value: "+hardwareSerialBaud)
				MainStatus.Log(logging.Warning(), "Disabling hardware serial functionality")
				Config.HardwareSerialEnabled = false
				return
			}

			HardwareSerialBaud = baudrateString
		}
		// Start initial reader / writer
		startSerialComms(HardwareSerialPort, HardwareSerialBaud)

		// Setup other devices
		for device, baudrate := range mserial.ParseSerialDevices(settingsData) {
			startSerialComms(device, baudrate)
		}
	}
}

// startSerialComms will set up the serial port,
// and start the ReadSerial goroutine
func startSerialComms(deviceName string, baudrate int) {
	MainStatus.Log(logging.OK(), "Opening serial device "+deviceName)
	c := &serial.Config{Name: deviceName, Baud: baudrate}
	s, err := serial.OpenPort(c)
	if err != nil {
		MainStatus.Log(logging.Error(), "Failed to open serial port "+deviceName)
		MainStatus.Log(logging.Error(), err.Error())
	} else {
		// Use first Serial device as a R/W, all others will only be read from
		if Config.SerialControlDevice == nil {
			Config.SerialControlDevice = s
			MainStatus.Log(logging.OK(), "Using serial device "+deviceName+" as default writer")
		}

		// Continiously read from serial port
		go mserial.ReadSerial(s, MainSession)
	}
}
