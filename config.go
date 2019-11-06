package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/influx"
	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
	"github.com/MrDoctorKovacic/MDroid-Core/pybus"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions/gps"
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
	log.Info().Msg("Starting MDroid Core")
	flag.StringVar(&settings.Settings.File, "settings-file", "", "File to recover the persistent settings.")
	debug := flag.Bool("debug", false, "sets log level to debug")
	flag.Parse()

	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// Parse settings file
	log.Info().Msg("Checking settings file...")
	settings.ReadFile(settings.Settings.File)

	// Check settings
	if _, err := json.Marshal(settings.GetAll()); err != nil {
		panic(err)
	}

	// Default video status
	sessions.SetValue("VIDEO_ON", "TRUE")

	// Setup hooks for extra settings/session parsing
	setupHooks()

	// Parse through config if found in settings file
	configMap, err := settings.GetComponent("MDROID")
	if err != nil {
		log.Warn().Msg("No config found in settings file, not parsing through config")
		return // abort config
	}

	gps.SetupTimezone(&configMap)
	setupDatabase(&configMap)
	sessions.SetupTokens(&configMap)
	setupSerial()

	// Set up pybus repeat commands
	if _, usingPybus := configMap["PYBUS_DEVICE"]; usingPybus {
		pybus.StartRepeats()
	}
}

// Set up InfluxDB time series logging
func setupDatabase(configAddr *map[string]string) {
	configMap := *configAddr
	databaseHost, usingDatabase := configMap["DATABASE_HOST"]
	if !usingDatabase {
		influx.DB = nil
		log.Info().Msg("Not logging to influx db")
		return
	}
	influx.DB = &influx.Influx{Host: databaseHost, Database: configMap["DATABASE_NAME"]}
}

func setupSerial() {
	configMap, err := settings.GetComponent("MDROID")
	if err != nil {
		log.Error().Msg(fmt.Sprintf("Failed to read MDROID settings. Not setting up serial devices.\n%s", err.Error()))
		return
	}
	hardwareSerialPort, usingHardwareSerial := configMap["HARDWARE_SERIAL_PORT"]

	if !usingHardwareSerial {
		log.Error().Msg(fmt.Sprintf("No hardware serial port. Not setting up serial devices.\n%s", err.Error()))
		return
	}

	// Start initial reader / writer
	go sessions.StartSerialComms(hardwareSerialPort, 9600)

	// Setup other devices
	for device, baudrate := range mserial.ParseSerialDevices(settings.GetAll()) {
		go sessions.StartSerialComms(device, baudrate)
	}
}
