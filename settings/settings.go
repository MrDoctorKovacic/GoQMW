package settings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/MrDoctorKovacic/MDroid-Core/external/status"
	"github.com/MrDoctorKovacic/MDroid-Core/influx"
	"github.com/gorilla/mux"
)

// settingsFile is the internal reference file for saving settings to
var settingsFile = "./settings.json"

// Settings control generic user defined field:value mappings, which will persist each run
var Settings map[string]map[string]string

// SettingsStatus will control logging and reporting of status / warnings / errors
var SettingsStatus = status.NewStatus("Settings")

// DB variable for influx
var DB influx.Influx
var databaseEnabled = false

// parseSettings will open and interpret program settings,
// as well as return the generic settings from last session
func parseSettings(settingsFile string) (map[string]map[string]string, error) {
	var data map[string]map[string]string

	// Open settings file
	file, err := os.Open(settingsFile)
	if err != nil {
		SettingsStatus.Log(status.Error(), "Error opening file '"+settingsFile+"': "+err.Error())
		return nil, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil {
		SettingsStatus.Log(status.Error(), "Error parsing json from file '"+settingsFile+"': "+err.Error())
		return nil, err
	}

	return data, nil
}

// Setup will handle the initialization of settings,
// either from past mapping or by creating a new one
func Setup(useSettingsFile string) map[string]map[string]string {
	if useSettingsFile != "" {
		settingsFile = useSettingsFile
		initSettings, err := parseSettings(settingsFile)
		if err == nil && initSettings != nil && len(initSettings) != 0 {
			Settings = initSettings

			// Log settings
			out, err := json.Marshal(Settings)
			if err == nil {
				SettingsStatus.Log(status.OK(), "Successfully loaded settings from file '"+settingsFile+"': "+string(out))
				return Settings
			}

			// If err is set, re-marshaling the settings failed
			SettingsStatus.Log(status.Warning(), "Failed to load settings from file '"+settingsFile+"'. Defaulting to empty Map. Error: "+err.Error())
		} else if initSettings == nil {
			SettingsStatus.Log(status.Warning(), "Failed to load settings from file '"+settingsFile+"'. Is it empty?")
		}
	}

	// Default to empty map
	Settings = make(map[string]map[string]string, 0)

	// Return empty map
	return Settings
}

// SetupDatabase is optional, but enables logging POST requests to see where things are coming from
func SetupDatabase(database influx.Influx) {
	DB = database
	databaseEnabled = true
}

// WriteSettings to given file, create one if it doesn't exist
func WriteSettings() error {
	settingsJSON, _ := json.Marshal(Settings)
	err := ioutil.WriteFile(settingsFile, settingsJSON, 0644)
	if err != nil {
		SettingsStatus.Log(status.Error(), "Failed to write Settings to "+settingsFile+": "+err.Error())
		return err
	}

	// Log success
	SettingsStatus.Log(status.OK(), "Successfully wrote Settings to "+settingsFile)
	return nil
}

// GetAllSettings returns all current settings
func GetAllSettings(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(Settings)
}

// GetSetting returns all the values of a specific setting
func GetSetting(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	json.NewEncoder(w).Encode(Settings[params["component"]])
}

// GetSettingValue returns a specific setting value
func GetSettingValue(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	json.NewEncoder(w).Encode(Settings[params["component"]][params["name"]])
}

// SetSettingValue updates or posts a new setting value
func SetSettingValue(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	// Insert component into Map if not exists
	if _, ok := Settings[params["component"]]; !ok {
		Settings[params["component"]] = make(map[string]string, 0)
	}

	// Update setting in inner map
	Settings[params["component"]][params["name"]] = params["value"]

	// Insert into database
	if databaseEnabled {
		// Format as string, if it is. Influx will complain about booleans / ints formatted as strings
		var value = params["value"]
		_, err := strconv.Atoi(params["value"])
		if err != nil {
			value = "\"" + params["value"] + "\""
		}

		err = DB.Write(fmt.Sprintf("settings,component=%s %s=%s", strings.Replace(params["component"], " ", "_", -1), params["name"], value))

		if err != nil {
			SettingsStatus.Log(status.Error(), fmt.Sprintf("Error writing %s[%s] = %s to influx DB: %s", params["component"], params["name"], params["value"], err.Error()))
		} else {
			SettingsStatus.Log(status.OK(), fmt.Sprintf("Logged %s[%s] = %s to database", params["component"], params["name"], params["value"]))
		}
	}

	// Log our success
	SettingsStatus.Log(status.OK(), fmt.Sprintf("Updated setting %s[%s] to %s", params["component"], params["name"], params["value"]))

	// Write out all settings to a file
	WriteSettings()

	// Respond with ack
	json.NewEncoder(w).Encode("OK")
}
