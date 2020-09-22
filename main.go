package main

import (
	"flag"

	"github.com/qcasey/MDroid-Core/internal/core"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Info().Msg("Starting MDroid Core")

	var settingsFile string
	flag.StringVar(&settingsFile, "settings-file", "", "File to recover the persistent settings.")
	flag.Parse()

	// Create new MDroid Core program
	core := core.New(settingsFile)
	addRoutes(core.Router)

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

	addCustomHooks()

	// Setup conventional modules
	//mserial.Setup(router)
	//bluetooth.Setup(router)
	//pybus.Setup(router)
	//db.Setup()

	// Start MDroid Core
	core.Start()
}
