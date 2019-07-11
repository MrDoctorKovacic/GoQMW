package settings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/external/influx"
	"github.com/MrDoctorKovacic/MDroid-Core/external/status"
	"github.com/gorilla/mux"
)

// settingsFile is the internal reference file for saving settings to
var settingsFile = "./settings.json"

// Settings control generic user defined field:value mappings, which will persist each run
// The mutex should be unnecessary, but is provided just in case
var Settings map[string]map[string]string
var settingsLock sync.Mutex

// SettingsStatus will control logging and reporting of status / warnings / errors
var SettingsStatus = status.NewStatus("Settings")

// verboseOutput will determine how much to put out to logs
var verboseOutput bool

// DB variable for influx
var DB influx.Influx
var databaseEnabled = false

// Formats in upper case with underscores replacing spaces
func formatSetting(text string) string {
	return strings.ToUpper(strings.Replace(text, " ", "_", -1))
}

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

// writeSettings to given file, create one if it doesn't exist
func writeSettings(file string) error {
	settingsLock.Lock()
	settingsJSON, err := json.Marshal(Settings)
	settingsLock.Unlock()

	if err != nil {
		SettingsStatus.Log(status.Error(), "Failed to marshall Settings")
		return err
	}

	err = ioutil.WriteFile(file, settingsJSON, 0644)
	if err != nil {
		SettingsStatus.Log(status.Error(), "Failed to write Settings to "+file+": "+err.Error())
		return err
	}

	// Log success
	SettingsStatus.Log(status.OK(), "Successfully wrote Settings to "+file)
	return nil
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

	if useSettingsFile != "" {
		SetSetting("CONFIG", "LAST_USED", time.Now().String())
	}

	// Return empty map
	return Settings
}

// SetupDatabase is optional, but enables logging POST requests to see where things are coming from
func SetupDatabase(database influx.Influx, isVerbose bool) {
	DB = database
	databaseEnabled = true
	verboseOutput = isVerbose
	if verboseOutput {
		SettingsStatus.Log(status.OK(), "Initialized Database for Settings")
	}
}

// GetAllSettings returns all current settings
func GetAllSettings(w http.ResponseWriter, r *http.Request) {
	if verboseOutput {
		SettingsStatus.Log(status.OK(), "Responding to GET request with entire settings map.")
	}
	json.NewEncoder(w).Encode(Settings)
}

// GetSetting returns all the values of a specific setting
func GetSetting(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	componentName := formatSetting(params["componentName"])

	if verboseOutput {
		SettingsStatus.Log(status.OK(), fmt.Sprintf("Responding to GET request for setting component %s", componentName))
	}

	json.NewEncoder(w).Encode(Settings[componentName])
}

// GetSettingValue returns a specific setting value
func GetSettingValue(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	componentName := formatSetting(params["componentName"])
	settingName := formatSetting(params["name"])

	if verboseOutput {
		SettingsStatus.Log(status.OK(), fmt.Sprintf("Responding to GET request for setting %s on component %s", settingName, componentName))
	}

	json.NewEncoder(w).Encode(Settings[componentName][settingName])
}

// SetSettingValue is the http wrapper for our setting setter
func SetSettingValue(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	// Parse out params
	componentName := formatSetting(params["componentName"])
	settingName := formatSetting(params["name"])
	settingValue := params["value"]

	// Log if requested
	if verboseOutput {
		SettingsStatus.Log(status.OK(), fmt.Sprintf("Responding to POST request for setting %s on component %s to be value %s", settingName, componentName, settingValue))
	}

	// Do the dirty work elsewhere
	SetSetting(componentName, settingName, settingValue)

	// Respond with ack
	json.NewEncoder(w).Encode("OK")
}

// SetSetting will handle actually updates or posts a new setting value
func SetSetting(componentName string, settingName string, settingValue string) {
	// Insert componentName into Map if not exists
	settingsLock.Lock()
	if _, ok := Settings[componentName]; !ok {
		Settings[componentName] = make(map[string]string, 0)
	}

	// Update setting in inner map
	Settings[componentName][settingName] = settingValue
	settingsLock.Unlock()

	// Insert into database
	if databaseEnabled {
		// Format as string, if it is. Influx will complain about booleans / ints formatted as strings
		var value = settingValue
		_, err := strconv.Atoi(settingValue)
		if err != nil {
			value = "\"" + settingValue + "\""
		}

		err = DB.Write(fmt.Sprintf("settings,component=%s %s=%s", componentName, settingName, value))

		if err != nil {
			SettingsStatus.Log(status.Error(), fmt.Sprintf("Error writing %s[%s] = %s to influx DB: %s", componentName, settingName, settingValue, err.Error()))
		} else {
			SettingsStatus.Log(status.OK(), fmt.Sprintf("Logged %s[%s] = %s to database", componentName, settingName, settingValue))
		}
	}

	// Log our success
	SettingsStatus.Log(status.OK(), fmt.Sprintf("Updated setting %s[%s] to %s", componentName, settingName, settingValue))

	// Write out all settings to a file
	writeSettings(settingsFile)
}
