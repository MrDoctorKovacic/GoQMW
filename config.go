package main

import (
	"encoding/json"
	"flag"
	"strconv"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/bluetooth"
	"github.com/MrDoctorKovacic/MDroid-Core/influx"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/MrDoctorKovacic/MDroid-Core/status"
)

// Config controls program settings and general persistent settings
type Config struct {
	DatabaseHost     string
	DatabaseName     string
	BluetoothAddress string
	PingHost         string
	DebugSessionFile string
}

// DatabaseEnabled - If we're using a database at all
var DatabaseEnabled bool

// UsingHardwareSerial - a gateway to an Arduino hooked to a set of relays
var UsingHardwareSerial bool

// HardwareSerialPort - port for accessing hardware serial controls
var HardwareSerialPort string

// HardwareSerialBaud - respective baud rate
var HardwareSerialBaud string

// DB for influx, that may or may not be used globally
var DB influx.Influx

func parseConfig() map[string]string {
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
	config, ok := settingsData["CONFIG"]
	if ok {

		// Set up timezone
		timezoneLocation, usingTimezone := config["Timezone"]
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
		databaseHost, usingDatabase := config["CORE_DATABASE_HOST"]
		if usingDatabase {
			DB = influx.Influx{Host: databaseHost, Database: config["CORE_DATABASE_NAME"]}
			DatabaseEnabled = true

			// Set up ping functionality
			// Proprietary pinging for component tracking
			if config["CORE_PING_HOST"] != "" {
				status.RemotePingAddress = config["CORE_PING_HOST"]
			} else {
				MainStatus.Log(status.OK(), "[DISABLED] Not forwarding pings to host")
			}

		} else {
			DatabaseEnabled = false
			MainStatus.Log(status.OK(), "[DISABLED] Not logging to influx db")
		}

		// Set up bluetooth
		bluetoothAddress, usingBluetooth := config["BLUETOOTH_ADDRESS"]
		if usingBluetooth {
			bluetooth.EnableAutoRefresh()
			bluetooth.SetAddress(bluetoothAddress)
		}

		//
		// PROPRIETARY
		// Configure hardware serials, should not be used outside my own config
		//
		HardwareSerialPort, UsingHardwareSerial := config["CORE_HardwareSerialPort"]
		hardwareSerialBaud, usingHardwareBaud := config["CORE_HardwareSerialBaud"]
		if UsingHardwareSerial {
			// Configure default baudrate
			HardwareSerialBaud := 9600
			if usingHardwareBaud {
				baudrateString, err := strconv.Atoi(hardwareSerialBaud)
				if err != nil {
					MainStatus.Log(status.Error(), "Failed to convert CORE_HardwareSerialBaud to int. Found value: "+hardwareSerialBaud)
					MainStatus.Log(status.Warning(), "Disabling hardware serial functionality")
					UsingHardwareSerial = false
				} else {
					HardwareSerialBaud = baudrateString
				}
			}
			StartSerialComms(HardwareSerialPort, HardwareSerialBaud)
		}

		return config
	}

	MainStatus.Log(status.Warning(), "No config found in settings file, not parsing through config")
	return nil
}
