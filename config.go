package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/pybus"

	"github.com/MrDoctorKovacic/MDroid-Core/bluetooth"
	"github.com/MrDoctorKovacic/MDroid-Core/influx"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/tarm/serial"
)

// ConfigValues controls program settings and general persistent settings
type ConfigValues struct {
	AuthToken             string
	DatabaseEnabled       bool
	DatabaseHost          string
	DatabaseName          string
	BluetoothAddress      string
	BluetoothEnabled      bool
	PingHost              string
	DebugSessionFile      string
	HardwareSerialEnabled bool
	HardwareSerialPort    string
	HardwareSerialBaud    string
	SerialControlDevice   *serial.Port
	VerboseOutput         bool
}

// Config defined here, to be saved to below
var Config ConfigValues

// DB for influx, that may or may not be used globally
var DB influx.Influx

// Main config parsing
func parseConfig() {

	// Start with program arguments
	var (
		settingsFile string
		sessionFile  string // This is for debugging ONLY
	)
	flag.StringVar(&settingsFile, "settings-file", "", "File to recover the persistent settings.")
	flag.StringVar(&sessionFile, "session-file", "", "[DEBUG ONLY] File to save and recover the last-known session.")
	flag.Parse()

	// Parse settings file
	settingsData, VerboseOutput := settings.SetupSettings(settingsFile)
	Config.VerboseOutput = VerboseOutput
	SetupSessions(sessionFile)

	// Log settings
	out, err := json.Marshal(settingsData)
	if err != nil {
		panic(err)
	}
	MainStatus.Log(logging.OK(), "Using settings: "+string(out))

	// Init session tracking (with or without Influx)
	// Fetch and append old session from disk if allowed

	// Parse through config if found in settings file
	configMap, ok := settingsData["MDROID"]
	if ok {

		// Set up timezone
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

		// Set up InfluxDB time series logging
		databaseHost, usingDatabase := configMap["DATABASE_HOST"]
		if usingDatabase {
			DB = influx.Influx{Host: databaseHost, Database: configMap["DATABASE_NAME"]}
			Config.DatabaseEnabled = true

			// Set up ping functionality
			// Proprietary pinging for component tracking
			if configMap["PING_HOST"] != "" {
				logging.RemotePingAddress = configMap["PING_HOST"]
			} else {
				MainStatus.Log(logging.OK(), "[DISABLED] Not forwarding pings to host")
			}

		} else {
			Config.DatabaseEnabled = false
			MainStatus.Log(logging.OK(), "[DISABLED] Not logging to influx db")
		}

		// Set up bluetooth
		bluetoothAddress, usingBluetooth := configMap["BLUETOOTH_ADDRESS"]
		if usingBluetooth {
			bluetooth.EnableAutoRefresh()
			bluetooth.SetAddress(bluetoothAddress)
			Config.BluetoothAddress = bluetoothAddress
		}
		Config.BluetoothEnabled = usingBluetooth

		// Set up pybus repeat commands
		_, usingPybus := configMap["PYBUS_DEVICE"]
		if usingPybus {
			go pybus.RepeatCommand("requestIgnitionStatus", 10)
			go pybus.RepeatCommand("requestLampStatus", 20)
			go pybus.RepeatCommand("requestVehicleStatus", 30)
			go pybus.RepeatCommand("requestOdometer", 45)
			go pybus.RepeatCommand("requestTimeStatus", 60)
			go pybus.RepeatCommand("requestTemperatureStatus", 120)
		}

		// Debug session log
		Config.DebugSessionFile = configMap["DEBUG_SESSION_LOG"]

		// Set up Auth tokens
		authToken, usingAuth := configMap["AUTH_TOKEN"]
		if usingAuth {
			Config.AuthToken = authToken
		}

		//
		// PROPRIETARY
		// Configure hardware serials, should not be used outside my own config
		//
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
	} else {
		MainStatus.Log(logging.Warning(), "No config found in settings file, not parsing through config")
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
