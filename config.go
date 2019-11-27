package main

import (
	"flag"
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
	settings.ReadFile(settings.Settings.File)

	// Parse through config if found in settings file
	configMap, err := settings.GetComponent("MDROID")
	if err != nil {
		log.Warn().Msg("MDROID settings not found, aborting config")
		return // abort config
	}

	// Enable debugging from settings
	if debuggingEnabled, ok := configMap["DEBUG"]; ok && debuggingEnabled == "TRUE" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	gps.SetupTimezone(&configMap)
	setupDatabase(&configMap)
	sessions.SetupTokens(&configMap)
	sessions.InitializeDefaults()
	setupSerial()
	setupHooks()

	// Set up pybus repeat commands
	if _, usingPybus := configMap["PYBUS_DEVICE"]; usingPybus {
		pybus.RunStartup()
		pybus.StartRepeats()
	}
	log.Info().Msg("Configuration complete, starting server...")
}

// Set up InfluxDB time series logging
func setupDatabase(configAddr *map[string]string) {
	configMap := *configAddr
	databaseHost, usingDatabase := configMap["DATABASE_HOST"]
	if !usingDatabase {
		influx.DB = nil
		log.Warn().Msg("InfluxDB is disabled")
		return
	}
	influx.DB = &influx.Influx{Host: databaseHost, Database: configMap["DATABASE_NAME"]}
	log.Info().Msgf("Using InfluxDB at %s", databaseHost)
}

func setupSerial() {
	configMap, err := settings.GetComponent("MDROID")
	if err != nil {
		log.Error().Msgf("Failed to read MDROID settings. Not setting up serial devices.\n%s", err.Error())
		return
	}

	hardwareSerialPort, usingHardwareSerial := configMap["HARDWARE_SERIAL_PORT"]
	if !usingHardwareSerial {
		log.Error().Msgf("No hardware serial port. Not setting up serial devices.\n%s", err.Error())
		return
	}

	// Check if serial is required for startup
	// This allows setting an initial state without incorrectly triggering hooks
	serialRequiredSetting, ok := configMap["SERIAL_STARTUP"]
	if ok && serialRequiredSetting == "TRUE" {
		// Serial is required for setup.
		// Open a port, set state to the output and immediately close for later concurrent reading
		s, err := sessions.OpenSerialPort(hardwareSerialPort, 9600)
		if err != nil {
			log.Error().Msg(err.Error())
		}
		sessions.ReadSerial(s)
		log.Info().Msg("Closing port for later reading")
		s.Close()
	}

	// Start initial reader / writer
	log.Info().Msgf("Registering %s as serial writer", hardwareSerialPort)
	go sessions.StartSerialComms(hardwareSerialPort, 9600)

	// Setup other devices
	for device, baudrate := range mserial.ParseSerialDevices(settings.GetAll()) {
		go sessions.StartSerialComms(device, baudrate)
	}
}
