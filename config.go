package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/bluetooth"
	"github.com/MrDoctorKovacic/MDroid-Core/influx"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/tarm/serial"
)

// ConfigValues controls program settings and general persistent settings
type ConfigValues struct {
	AuthToken             string
	BluetoothAddress      string
	BluetoothEnabled      bool
	DB                    *influx.Influx
	DatabaseEnabled       bool
	DatabaseHost          string
	DatabaseName          string
	DebugSessionFile      string
	HardwareSerialEnabled bool
	HardwareSerialPort    string
	HardwareSerialBaud    string
	PingHost              string
	SerialControlDevice   *serial.Port
	SettingsFile          string
	SessionFile           string
	SlackURL              string
	VerboseOutput         bool
}

// Config defined here, to be saved to below
var Config ConfigValues

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
	SetupSessions(Config.SettingsFile)

	// Parse through config if found in settings file
	configMap, ok := settingsData["MDROID"]
	if ok {

		setupTimezone(&configMap)
		setupDatabase(&configMap)
		setupBluetooth(&configMap)

		// Set up pybus repeat commands
		_, usingPybus := configMap["PYBUS_DEVICE"]
		if usingPybus {
			go RepeatCommand("requestIgnitionStatus", 10)
			go RepeatCommand("requestLampStatus", 20)
			go RepeatCommand("requestVehicleStatus", 30)
			go RepeatCommand("requestOdometer", 45)
			go RepeatCommand("requestTimeStatus", 60)
			go RepeatCommand("requestTemperatureStatus", 120)
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
			Timezone = loc
		} else {
			// If timezone has errored
			Timezone, _ = time.LoadLocation("UTC")
		}
	} else {
		// If timezone is not set in config
		Timezone, _ = time.LoadLocation("UTC")
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
		StartSerialComms(HardwareSerialPort, HardwareSerialBaud)

		// Setup other devices
		parseSerialDevices(settingsData)
	}
}

// AuthMiddleware will match http bearer token again the one hardcoded in our config
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		reqToken := r.Header.Get("Authorization")
		splitToken := strings.Split(reqToken, "Bearer")
		if len(splitToken) != 2 || strings.TrimSpace(splitToken[1]) != Config.AuthToken {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403 - Invalid Auth Token!"))
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}
