package main

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/bluetooth"
	"github.com/qcasey/MDroid-Core/db"
	"github.com/qcasey/MDroid-Core/mqtt"
	"github.com/qcasey/MDroid-Core/mserial"
	"github.com/qcasey/MDroid-Core/pybus"
	"github.com/qcasey/MDroid-Core/sessions"
	"github.com/qcasey/MDroid-Core/sessions/gps"
	"github.com/qcasey/MDroid-Core/settings"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func configureLogging(debug *bool) {
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
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

func main() {
	log.Info().Msg("Starting MDroid Core")

	var settingsFile string
	flag.StringVar(&settingsFile, "settings-file", "", "File to recover the persistent settings.")
	debug := flag.Bool("debug", false, "sets log level to debug")
	flag.Parse()
	configureLogging(debug)

	settings.ParseConfig(settingsFile)

	sessions.Setup()
	setupHooks()

	// Init router
	router := mux.NewRouter()

	gps.Mod.Setup()
	gps.Mod.SetRoutes(router)

	// Set default routes (including session)
	SetDefaultRoutes(router)

	// Setup conventional modules
	// TODO: More modular handling of modules
	mserial.Mod.Setup()
	mserial.Mod.SetRoutes(router)
	bluetooth.Mod.Setup()
	bluetooth.Mod.SetRoutes(router)
	pybus.Mod.Setup()
	pybus.Mod.SetRoutes(router)
	db.Mod.Setup()
	mqtt.Mod.Setup()

	// Connect bluetooth device on startup
	bluetooth.Connect()

	Start(router)
}
