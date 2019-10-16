package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strconv"

	"github.com/MrDoctorKovacic/MDroid-Core/bluetooth"
	"github.com/MrDoctorKovacic/MDroid-Core/gps"
	"github.com/MrDoctorKovacic/MDroid-Core/influx"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
	"github.com/MrDoctorKovacic/MDroid-Core/pybus"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/rs/zerolog"
)

// Main config parsing
func parseConfig() {

	var sessionFile string
	flag.StringVar(&settings.Settings.File, "settings-file", "", "File to recover the persistent settings.")
	flag.StringVar(&sessionFile, "session-file", "", "[DEBUG ONLY] File to save and recover the last-known session.")
	debug := flag.Bool("debug", false, "sets log level to debug")
	flag.Parse()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// Setup hooks for extra settings/session parsing
	setupHooks()

	// Parse settings file
	settings.ReadFile(settings.Settings.File)

	// Check settings
	if _, err := json.Marshal(settings.GetAll()); err != nil {
		panic(err)
	}

	// Init session tracking (with or without Influx)
	sessions.Create(settings.Settings.File)

	// Parse through config if found in settings file
	configMap, ok := settings.GetAll()["MDROID"]
	if !ok {
		mainStatus.Log(logging.Warning(), "No config found in settings file, not parsing through config")
	}

	gps.SetupTimezone(&configMap)
	setupDatabase(&configMap)
	bluetooth.Setup(&configMap)
	sessions.SetupTokens(&configMap)
	setupSerial()

	settings.Config.SlackURL = configMap["SLACK_URL"]

	// Set up pybus repeat commands
	if _, usingPybus := configMap["PYBUS_DEVICE"]; usingPybus {
		go pybus.RepeatCommand("requestIgnitionStatus", 10)
		go pybus.RepeatCommand("requestLampStatus", 20)
		go pybus.RepeatCommand("requestVehicleStatus", 30)
		go pybus.RepeatCommand("requestOdometer", 45)
		go pybus.RepeatCommand("requestTimeStatus", 60)
		go pybus.RepeatCommand("requestTemperatureStatus", 120)
	}
}

// Set up InfluxDB time series logging
func setupDatabase(configAddr *map[string]string) {
	configMap := *configAddr
	databaseHost, usingDatabase := configMap["DATABASE_HOST"]
	if !usingDatabase {
		settings.Config.DB = nil
		mainStatus.Log(logging.OK(), "Not logging to influx db")
		return
	}
	settings.Config.DB = &influx.Influx{Host: databaseHost, Database: configMap["DATABASE_NAME"]}

	// Set up ping functionality
	// Proprietary pinging for component tracking
	if configMap["PING_HOST"] == "" {
		mainStatus.Log(logging.OK(), "Not forwarding pings to host")
		return
	}
	logging.RemotePingAddress = configMap["PING_HOST"]
}

//
// PROPRIETARY
// Configure hardware serials, should not be used outside my own config
//

func setupSerial() {
	configMap, err := settings.GetComponent("MDROID")
	if err != nil {
		mainStatus.Log(logging.Error(), fmt.Sprintf("Failed to read MDROID settings. Not setting up serial devices.\n%s", err.Error()))
		return
	}
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
		go sessions.StartSerialComms(HardwareSerialPort, HardwareSerialBaud)

		// Setup other devices
		for device, baudrate := range mserial.ParseSerialDevices(settings.GetAll()) {
			go sessions.StartSerialComms(device, baudrate)
		}
	}
}
