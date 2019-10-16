package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/bluetooth"
	"github.com/MrDoctorKovacic/MDroid-Core/gps"
	"github.com/MrDoctorKovacic/MDroid-Core/influx"
	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
	"github.com/MrDoctorKovacic/MDroid-Core/pybus"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().In(gps.GetTimezone())
	}
	zerolog.CallerMarshalFunc = func(file string, line int) string {
		fileparts := strings.Split(file, "/")
		filename := strings.Replace(fileparts[len(fileparts)-1], ".go", "", -1)
		return filename + ":" + strconv.Itoa(line)
	}
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "Mon Jan 2 15:04:05"}
	log.Logger = zerolog.New(output).With().Caller().Timestamp().Logger()
}

// Main config parsing
func parseConfig() {
	flag.StringVar(&settings.Settings.File, "settings-file", "", "File to recover the persistent settings.")
	debug := flag.Bool("debug", false, "sets log level to debug")
	flag.Parse()

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

	// Parse through config if found in settings file
	configMap, err := settings.GetComponent("MDROID")
	if err != nil {
		log.Warn().Msg("No config found in settings file, not parsing through config")
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
		log.Info().Msg("Not logging to influx db")
		return
	}
	settings.Config.DB = &influx.Influx{Host: databaseHost, Database: configMap["DATABASE_NAME"]}
}

//
// PROPRIETARY
// Configure hardware serials, should not be used outside my own config
//

func setupSerial() {
	configMap, err := settings.GetComponent("MDROID")
	if err != nil {
		log.Error().Msg(fmt.Sprintf("Failed to read MDROID settings. Not setting up serial devices.\n%s", err.Error()))
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
				log.Error().Msg("Failed to convert HardwareSerialBaud to int. Found value: " + hardwareSerialBaud)
				log.Warn().Msg("Disabling hardware serial functionality")
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
