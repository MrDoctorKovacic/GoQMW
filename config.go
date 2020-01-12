package main

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	bluetooth "github.com/qcasey/MDroid-Bluetooth"
	"github.com/qcasey/MDroid-Core/db"
	"github.com/qcasey/MDroid-Core/mserial"
	"github.com/qcasey/MDroid-Core/pybus"
	"github.com/qcasey/MDroid-Core/sessions"
	"github.com/qcasey/MDroid-Core/sessions/gps"
	"github.com/qcasey/MDroid-Core/settings"
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
func parseConfig() *mux.Router {
	log.Info().Msg("Starting MDroid Core")

	// Init router
	router := mux.NewRouter()

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
		return router // abort config
	}

	// Enable debugging from settings
	if debuggingEnabled, ok := configMap["DEBUG"]; ok && debuggingEnabled == "TRUE" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	gps.SetupTimezone(&configMap)
	setupDatabase(&configMap)
	sessions.Setup(&configMap)
	setupSerial(&configMap)

	// Setup conventional modules
	bluetooth.Setup(&configMap)
	bluetooth.SetRoutes(router)

	setupHooks()

	// Set up pybus repeat commands
	go func() {
		time.Sleep(500)
		if _, usingPybus := configMap["PYBUS_DEVICE"]; usingPybus {
			pybus.RunStartup()
			pybus.StartRepeats()
		}
	}()

	log.Info().Msg("Configuration complete, starting server...")
	return router
}

// Set up time series logging
func setupDatabase(configAddr *map[string]string) {
	configMap := *configAddr
	databaseHost, usingDatabase := configMap["DATABASE_HOST"]
	if !usingDatabase {
		db.DB = nil
		log.Warn().Msg("Databases are disabled")
		return
	}

	databaseName, usingDatabase := configMap["DATABASE_NAME"]
	if !usingDatabase {
		db.DB = nil
		log.Warn().Msg("Databases are disabled")
		return
	}

	// Request to use SQLITE
	if databaseHost == "SQLITE" {
		db.DB = &db.Database{Host: databaseHost, DatabaseName: databaseName, Type: db.SQLite}
		dbname, err := db.DB.SQLiteInit()
		if err != nil {
			panic(err)
		}
		log.Info().Msgf("Using SQLite DB at %s", dbname)
		return
	}

	// Setup InfluxDB as normal
	db.DB = &db.Database{Host: databaseHost, DatabaseName: databaseName, Type: db.InfluxDB}
	log.Info().Msgf("Using InfluxDB at %s with DB name %s.", databaseHost, databaseName)
}

func setupSerial(configAddr *map[string]string) {
	configMap := *configAddr

	hardwareSerialPort, usingHardwareSerial := configMap["HARDWARE_SERIAL_PORT"]
	if !usingHardwareSerial {
		log.Warn().Msgf("No hardware serial port defined. Not setting up serial devices.")
		return
	}

	// Check if serial is required for startup
	// This allows setting an initial state without incorrectly triggering hooks
	serialRequiredSetting, ok := configMap["SERIAL_STARTUP"]
	if ok && serialRequiredSetting == "TRUE" {
		// Serial is required for setup.
		// Open a port, set state to the output and immediately close for later concurrent reading
		s, err := sessions.OpenSerialPort(hardwareSerialPort, 115200)
		if err != nil {
			log.Error().Msg(err.Error())
		}
		sessions.ReadSerial(s)
		log.Info().Msg("Closing port for later reading")
		s.Close()
	}

	// Start initial reader / writer
	log.Info().Msgf("Registering %s as serial writer", hardwareSerialPort)
	go sessions.StartSerialComms(hardwareSerialPort, 115200)

	// Setup other devices
	for device, baudrate := range mserial.ParseSerialDevices(settings.GetAll()) {
		go sessions.StartSerialComms(device, baudrate)
	}
}
