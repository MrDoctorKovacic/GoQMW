package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"strconv"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/bluetooth"
	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/gps"
	"github.com/MrDoctorKovacic/MDroid-Core/influx"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/gorilla/mux"
	"github.com/tarm/serial"
)

// Config defined here, to be saved to below
//var Config settings.ConfigValues

// MainSession object
var MainSession *sessions.Session

// Main config parsing
func parseConfig() {

	var sessionFile string
	flag.StringVar(&settings.Config.SettingsFile, "settings-file", "", "File to recover the persistent settings.")
	flag.StringVar(&sessionFile, "session-file", "", "[DEBUG ONLY] File to save and recover the last-known session.")
	flag.Parse()

	// Parse settings file
	settings.ReadFile(settings.Config.SettingsFile)

	// Check settings
	if _, err := json.Marshal(settings.Data); err != nil {
		panic(err)
	}

	// Init session tracking (with or without Influx)
	// Fetch and append old session from disk if allowed
	MainSession = sessions.CreateSession(settings.Config.SettingsFile)
	//MainSession.Config = &Config
	MainSession.File = sessionFile

	// Parse through config if found in settings file
	configMap, ok := settings.GetAll()["MDROID"]
	if ok {
		setupTimezone(configMap)
		setupDatabase(configMap)
		setupBluetooth(configMap)
		setupTokens(configMap)
		setupSerial()

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

		// Slack URL
		settings.Config.SlackURL = configMap["SLACK_URL"]

	} else {
		mainStatus.Log(logging.Warning(), "No config found in settings file, not parsing through config")
	}
}

func setupTokens(configMap map[string]string) {
	// Set up Auth tokens
	token, usingTokens := configMap["AUTH_TOKEN"]
	serverHost, usingCentralHost := configMap["MDROID_SERVER"]

	if usingTokens && usingCentralHost {
		go MainSession.CheckServer(serverHost, token)
	} else {
		mainStatus.Log(logging.Warning(), "Missing central host parameters - checking into central host has been disabled. Are you sure this is correct?")
	}
}

func setupTimezone(configMap map[string]string) {
	settings.Config.Location = &gps.Location{}
	timezoneLocation, usingTimezone := configMap["Timezone"]
	if usingTimezone {
		loc, err := time.LoadLocation(timezoneLocation)
		if err == nil {
			settings.Config.Location.Timezone = loc
		} else {
			// If timezone has errored
			settings.Config.Location.Timezone, _ = time.LoadLocation("UTC")
		}
	} else {
		// If timezone is not set in config
		settings.Config.Location.Timezone, _ = time.LoadLocation("UTC")
	}
}

// Set up InfluxDB time series logging
func setupDatabase(configMap map[string]string) {
	databaseHost, usingDatabase := configMap["DATABASE_HOST"]
	if usingDatabase {
		settings.Config.DB = &influx.Influx{Host: databaseHost, Database: configMap["DATABASE_NAME"]}

		// Set up ping functionality
		// Proprietary pinging for component tracking
		if configMap["PING_HOST"] != "" {
			logging.RemotePingAddress = configMap["PING_HOST"]
		} else {
			mainStatus.Log(logging.OK(), "Not forwarding pings to host")
		}

	} else {
		settings.Config.DB = nil
		mainStatus.Log(logging.OK(), "Not logging to influx db")
	}
}

func setupBluetooth(configMap map[string]string) {
	bluetoothAddress, usingBluetooth := configMap["BLUETOOTH_ADDRESS"]
	if usingBluetooth {
		bluetooth.EnableAutoRefresh()
		bluetooth.SetAddress(bluetoothAddress)
		settings.Config.BluetoothAddress = bluetoothAddress
	}
	settings.Config.BluetoothAddress = ""
}

//
// PROPRIETARY
// Configure hardware serials, should not be used outside my own config
//

// WriteSerialHandler handles messages sent through the server
func WriteSerialHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	if params["command"] != "" {
		mserial.WriteSerial(settings.Config.SerialControlDevice, params["command"])
	}
	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "OK", Status: "success", OK: true})
}

func setupSerial() {
	configMap := settings.Data["MDROID"]
	HardwareSerialPort, usingHardwareSerial := configMap["HARDWARE_SERIAL_PORT"]
	hardwareSerialBaud, usingHardwareBaud := configMap["HARDWARE_SERIAL_BAUD"]
	settings.Config.HardwareSerialEnabled = usingHardwareSerial

	if settings.Config.HardwareSerialEnabled {
		// Configure default baudrate
		HardwareSerialBaud := 9600
		if usingHardwareBaud {
			baudrateString, err := strconv.Atoi(hardwareSerialBaud)
			if err != nil {
				mainStatus.Log(logging.Error(), "Failed to convert HardwareSerialBaud to int. Found value: "+hardwareSerialBaud)
				mainStatus.Log(logging.Warning(), "Disabling hardware serial functionality")
				settings.Config.HardwareSerialEnabled = false
				return
			}

			HardwareSerialBaud = baudrateString
		}
		// Start initial reader / writer
		startSerialComms(HardwareSerialPort, HardwareSerialBaud)

		// Setup other devices
		for device, baudrate := range mserial.ParseSerialDevices(settings.Data) {
			startSerialComms(device, baudrate)
		}
	}
}

// startSerialComms will set up the serial port,
// and start the ReadSerial goroutine
func startSerialComms(deviceName string, baudrate int) {
	mainStatus.Log(logging.OK(), "Opening serial device "+deviceName)
	c := &serial.Config{Name: deviceName, Baud: baudrate}
	s, err := serial.OpenPort(c)
	if err != nil {
		mainStatus.Log(logging.Error(), "Failed to open serial port "+deviceName)
		mainStatus.Log(logging.Error(), err.Error())
	} else {
		// Use first Serial device as a R/W, all others will only be read from
		if settings.Config.SerialControlDevice == nil {
			settings.Config.SerialControlDevice = s
			mainStatus.Log(logging.OK(), "Using serial device "+deviceName+" as default writer")
		}

		// Continiously read from serial port
		go MainSession.ReadFromSerial(s)
	}
}
