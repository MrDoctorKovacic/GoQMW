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

	setupDatabase(&configMap)
	sessions.Setup(&configMap)

	// Setup conventional modules
	mserial.Mod.Setup(&configMap)
	mserial.Mod.SetRoutes(router)
	bluetooth.Mod.Setup(&configMap)
	bluetooth.Mod.SetRoutes(router)
	gps.Mod.Setup(&configMap)
	gps.Mod.SetRoutes(router)

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
