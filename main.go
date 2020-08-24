package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/bluetooth"
	"github.com/qcasey/MDroid-Core/db"
	"github.com/qcasey/MDroid-Core/mqtt"
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

	settings.HL.RegisterHook("AUTO_SLEEP", autoSleepSettings)
	settings.HL.RegisterHook("AUTO_LOCK", autoLockSettings)
	settings.HL.RegisterHook("ANGEL_EYES", angelEyesSettings)
	sessions.HL.RegisterHook("ACC_POWER", accPower)
	sessions.HL.RegisterHook("LIGHT_SENSOR_REASON", lightSensorReason)
	sessions.HL.RegisterHook("LIGHT_SENSOR_ON", lightSensorOn)
	sessions.HL.RegisterHook("SEAT_MEMORY_1", seatMemory)
	sessions.HL.RegisterHook("ACC_POWER", accPower)
	log.Info().Msg(fmt.Sprintf("Enabled %d session hooks", sessions.HL.Length()))

	// Init router
	router := mux.NewRouter()

	// Set default routes (including session)
	SetDefaultRoutes(router)

	// Setup conventional modules
	// TODO: More modular handling of modules
	mserial.Setup(router)
	bluetooth.Setup(router)
	pybus.Setup(router)
	db.Setup()

	// Set up MQTT, more dependent than other packages
	if !settings.Data.IsSet("mdroid.MQTT_ADDRESS") || !settings.Data.IsSet("mdroid.MQTT_ADDRESS_FALLBACK") || !settings.Data.IsSet("mdroid.MQTT_CLIENT_ID") || !settings.Data.IsSet("mdroid.MQTT_USERNAME") || !settings.Data.IsSet("mdroid.MQTT_PASSWORD") {
		log.Warn().Msgf("Missing MQTT setup variables, skipping MQTT.")
	} else {
		mqtt.Setup(settings.Data.GetString("mdroid.MQTT_ADDRESS"), settings.Data.GetString("mdroid.MQTT_ADDRESS_FALLBACK"), settings.Data.GetString("mdroid.MQTT_CLIENT_ID"), settings.Data.GetString("mdroid.MQTT_USERNAME"), settings.Data.GetString("mdroid.MQTT_PASSWORD"))
	}

	Start(router)
}
