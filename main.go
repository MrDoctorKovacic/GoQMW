package main

import (
	"flag"

	"github.com/qcasey/MDroid-Core/internal/server"
	"github.com/qcasey/MDroid-Core/routes/serial"
	"github.com/qcasey/MDroid-Core/routes/shutdown"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Info().Msg("Starting MDroid Core")

	var settingsFile string
	flag.StringVar(&settingsFile, "settings-file", "", "File to recover the persistent settings.")
	flag.Parse()

	// Create new MDroid Core program
	srv := server.New(settingsFile)
	addRoutes(srv)

	//settings.ParseConfig(settingsFile)
	//sessions.Setup()

	/*
		flushToMQTT := settings.IsSet("mdroid.mqtt_address")
		if flushToMQTT {
			log.Info().Msg("Setting up MQTT")
			// Set up MQTT, more dependent than other packages
			if !settings.IsSet("mdroid.MQTT_ADDRESS") || !settings.IsSet("mdroid.MQTT_ADDRESS_FALLBACK") || !settings.IsSet("mdroid.MQTT_CLIENT_ID") || !settings.IsSet("mdroid.MQTT_USERNAME") || !settings.IsSet("mdroid.MQTT_PASSWORD") {
				log.Warn().Msgf("Missing MQTT setup variables, skipping MQTT.")
			} else {
				mqtt.Setup(settings.GetString("mdroid.MQTT_ADDRESS"), settings.GetString("mdroid.MQTT_ADDRESS_FALLBACK"), settings.GetString("mdroid.MQTT_CLIENT_ID"), settings.GetString("mdroid.MQTT_USERNAME"), settings.GetString("mdroid.MQTT_PASSWORD"))
			}
		}

		// Run hooks on all new settings
		settings := Data.AllKeys()
		log.Info().Msg("Settings:")
		for _, key := range settings {
			log.Info().Msgf("\t%s = %s", key, Data.GetString(key))
			if flushToMQTT {
				go mqtt.Publish(fmt.Sprintf("settings/%s", key), Data.GetString(key), true)
			}
			HL.RunHooks(key, Data.GetString(key))
		}*/

	//addCustomHooks()

	// Setup conventional modules
	//mserial.Setup(router)
	//bluetooth.Setup(router)
	//pybus.Setup(router)
	//db.Setup()

	// Start MDroid Core
	srv.Start()
}

// addRoutes initializes an MDroid router with default system routes
func addRoutes(srv *server.Server) {
	log.Info().Msg("Configuring module routes...")

	//
	// Module Routes
	//
	srv.Router.HandleFunc("/shutdown", shutdown.Shutdown(srv.Core)).Methods("GET")
	srv.Router.HandleFunc("/serial/{command}", serial.WriteSerial(srv.Core)).Methods("POST", "GET")
}
