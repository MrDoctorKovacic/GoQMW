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
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/MrDoctorKovacic/MDroid-Core/status"
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
}

// Config defined here, to be saved to below
var Config ConfigValues

// DB for influx, that may or may not be used globally
var DB influx.Influx

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
	settingsData, useVerboseOutput := settings.SetupSettings(settingsFile)
	VerboseOutput = useVerboseOutput
	SetupSessions(sessionFile)

	// Log settings
	out, err := json.Marshal(settingsData)
	if err != nil {
		panic(err)
	}
	MainStatus.Log(status.OK(), "Using settings: "+string(out))

	// Init session tracking (with or without Influx)
	// Fetch and append old session from disk if allowed

	// Parse through config if found in settings file
	configMap, ok := settingsData["CONFIG"]
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
		databaseHost, usingDatabase := configMap["CORE_DATABASE_HOST"]
		if usingDatabase {
			DB = influx.Influx{Host: databaseHost, Database: configMap["CORE_DATABASE_NAME"]}
			Config.DatabaseEnabled = true

			// Set up ping functionality
			// Proprietary pinging for component tracking
			if configMap["CORE_PING_HOST"] != "" {
				status.RemotePingAddress = configMap["CORE_PING_HOST"]
			} else {
				MainStatus.Log(status.OK(), "[DISABLED] Not forwarding pings to host")
			}

		} else {
			Config.DatabaseEnabled = false
			MainStatus.Log(status.OK(), "[DISABLED] Not logging to influx db")
		}

		// Set up bluetooth
		bluetoothAddress, usingBluetooth := configMap["BLUETOOTH_ADDRESS"]
		if usingBluetooth {
			bluetooth.EnableAutoRefresh()
			bluetooth.SetAddress(bluetoothAddress)
			Config.BluetoothAddress = bluetoothAddress
		}
		Config.BluetoothEnabled = usingBluetooth

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
		HardwareSerialPort, usingHardwareSerial := configMap["CORE_HARDWARE_SERIAL_PORT"]
		hardwareSerialBaud, usingHardwareBaud := configMap["CORE_HARDWARE_SERIAL_BAUD"]
		Config.HardwareSerialEnabled = usingHardwareSerial
		if Config.HardwareSerialEnabled {
			// Configure default baudrate
			HardwareSerialBaud := 9600
			if usingHardwareBaud {
				baudrateString, err := strconv.Atoi(hardwareSerialBaud)
				if err != nil {
					MainStatus.Log(status.Error(), "Failed to convert CORE_HardwareSerialBaud to int. Found value: "+hardwareSerialBaud)
					MainStatus.Log(status.Warning(), "Disabling hardware serial functionality")
					Config.HardwareSerialEnabled = false
				} else {
					HardwareSerialBaud = baudrateString
				}
			}
			StartSerialComms(HardwareSerialPort, HardwareSerialBaud)
		}
	}

	MainStatus.Log(status.Warning(), "No config found in settings file, not parsing through config")
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
