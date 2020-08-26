package main

import (
	"flag"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/bluetooth"
	"github.com/qcasey/MDroid-Core/db"
	"github.com/qcasey/MDroid-Core/mserial"
	"github.com/qcasey/MDroid-Core/pybus"
	"github.com/qcasey/MDroid-Core/sessions"
	"github.com/qcasey/MDroid-Core/settings"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func configureLogging(debug *bool) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.CallerMarshalFunc = func(file string, line int) string {
		fileparts := strings.Split(file, "/")
		filename := strings.Replace(fileparts[len(fileparts)-1], ".go", "", -1)
		return filename + ":" + strconv.Itoa(line)
	}
	zerolog.TimeFieldFormat = "3:04PM"
	output := zerolog.ConsoleWriter{Out: os.Stderr}
	log.Logger = zerolog.New(output).With().Timestamp().Caller().Logger()
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

	addCustomHooks()

	// Init router
	router := mux.NewRouter()

	// Setup conventional modules
	mserial.Setup(router)
	bluetooth.Setup(router)
	pybus.Setup(router)

	// Set default routes (including session)
	SetDefaultRoutes(router)

	db.Setup()

	Start(router)
}
