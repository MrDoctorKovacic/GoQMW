package main

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"time"

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
func parseConfig() *map[string]string {
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
		return &configMap // abort config
	}

	// Enable debugging from settings
	if debuggingEnabled, ok := configMap["DEBUG"]; ok && debuggingEnabled == "TRUE" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	sessions.Setup(&configMap)
	setupHooks()

	log.Info().Msg("Configuration complete.")
	return &configMap
}
