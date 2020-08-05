// Package settings reads and writes to an MDroid settings file
package settings

import (
	"fmt"
	"net/http"

	"github.com/qcasey/MDroid-Core/format"
	"github.com/qcasey/MDroid-Core/format/response"
	"github.com/qcasey/MDroid-Core/mqtt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/gorilla/mux"
)

// Setting is GraphQL handler struct
type Setting struct {
	Name        string `json:"name,omitempty"`
	Value       string `json:"value,omitempty"`
	LastUpdated string `json:"lastUpdated,omitempty"`
}

// Component is GraphQL handler struct
type Component struct {
	Name     string    `json:"name,omitempty"`
	Settings []Setting `json:"settings,omitempty"`
}

// ParseConfig will take initial configuration values and parse them into global settings
func ParseConfig(settingsFile string) {
	viper.SetConfigName(settingsFile) // name of config file (without extension)
	viper.AddConfigPath(".")          // optionally look for config in the working directory
	err := viper.ReadInConfig()       // Find and read the config file
	if err != nil {
		log.Warn().Msg(err.Error())
	}
	viper.WatchConfig()

	// Enable debugging from settings
	if viper.IsSet("mdroid.debug") && viper.GetBool("mdroid.debug") {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// Check if MQTT has an address and will be setup
	flushToMQTT := viper.GetString("mdroid.mqtt_address") != ""

	// Run hooks on all new settings
	settings := viper.AllSettings()
	for key := range settings {
		value := settings[key]
		if flushToMQTT {
			topic := fmt.Sprintf("settings/%s", key)
			go mqtt.Publish(topic, value, true)
		}
		runHooks(key, value)
	}
}

// HandleGetAll returns all current settings
func HandleGetAll(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msg("Responding to GET request with entire settings map.")
	resp := response.JSONResponse{Output: viper.AllSettings(), Status: "success", OK: true}
	resp.Write(&w, r)
}

// HandleGet returns all the values of a specific setting
func HandleGet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	componentName := format.Name(params["component"])

	log.Debug().Msgf("Responding to GET request for setting component %s", componentName)

	resp := response.JSONResponse{Output: viper.Get(params["component"]), OK: true}
	if !viper.IsSet(params["component"]) {
		resp = response.JSONResponse{Output: "Setting not found.", OK: false}
	}

	resp.Write(&w, r)
}

// HandleSet is the http wrapper for our setting setter
func HandleSet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	// Parse out params
	key := params["key"]
	value := params["value"]

	// Log if requested
	log.Debug().Msgf("Responding to POST request for setting %s on component %s to be value %s", key, value)

	// Do the dirty work elsewhere
	Set(key, value)

	// Respond with OK
	response := response.JSONResponse{Output: key, OK: true}
	response.Write(&w, r)
}

// Set will handle actually updates or posts a new setting value
func Set(key string, value interface{}) error {
	viper.Set(key, value)

	// Post to MQTT
	topic := fmt.Sprintf("settings/%s", key)
	go mqtt.Publish(topic, value, true)

	// Log our success
	log.Info().Msgf("Updated setting of %s to %s", key, value)

	viper.WriteConfig()

	// Trigger hooks
	runHooks(key, value)

	return nil
}

// Get will check if the given key exists, if not it will create it with the provided value
func Get(key string, value interface{}) interface{} {
	if !viper.IsSet(key) {
		Set(key, value)
	}
	return viper.Get(key)
}
