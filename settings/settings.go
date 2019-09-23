// Package settings reads and writes to an MDroid settings file
package settings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/gorilla/mux"
)

// settingsFile is the internal reference file for saving settings to
var settingsFile = "./settings.json"

// Settings control generic user defined field:value mappings, which will persist each run
// The mutex should be unnecessary, but is provided just in case
var Settings map[string]map[string]string
var settingsLock sync.Mutex

// SettingsStatus will control logging and reporting of status / warnings / errors
var SettingsStatus = logging.NewStatus("Settings")

// Configure verbose output
var verboseOutput bool

// ReadFile will handle the initialization of settings,
// either from past mapping or by creating a new one
func ReadFile(useSettingsFile string) (map[string]map[string]string, bool) {
	// Default to false
	verboseOutput = false

	if useSettingsFile != "" {
		settingsFile = useSettingsFile
		initSettings, err := parseFile(settingsFile)
		if err == nil && initSettings != nil && len(initSettings) != 0 {
			Settings = initSettings

			//
			// Check if we're configed to verbose output
			//
			var verboseOutputInt int
			useVerboseOutput, ok := Settings["MDROID"]["VERBOSE_OUTPUT"]
			if !ok {
				verboseOutputInt = 0
			} else {
				verboseOutputInt, err = strconv.Atoi(useVerboseOutput)
				if err != nil {
					verboseOutputInt = 0
				}
			}

			// Set as bool for return
			verboseOutput = verboseOutputInt != 0

			// Log settings
			out, err := json.Marshal(Settings)
			if err == nil {
				SettingsStatus.Log(logging.OK(), "Successfully loaded settings from file '"+settingsFile+"': "+string(out))
				return Settings, verboseOutput
			}

			// If err is set, re-marshaling the settings failed
			SettingsStatus.Log(logging.Warning(), "Failed to load settings from file '"+settingsFile+"'. Defaulting to empty Map. Error: "+err.Error())
		} else if initSettings == nil {
			SettingsStatus.Log(logging.Warning(), "Failed to load settings from file '"+settingsFile+"'. Is it empty?")
		}
	}

	// Default to empty map
	Settings = make(map[string]map[string]string, 0)

	if useSettingsFile != "" {
		Set("MDROID", "LAST_USED", time.Now().String())
	}

	// Return empty map
	return Settings, verboseOutput
}

// parseFile will open and interpret program settings,
// as well as return the generic settings from last session
func parseFile(settingsFile string) (map[string]map[string]string, error) {
	var data map[string]map[string]string

	// Open settings file
	file, err := os.Open(settingsFile)
	if err != nil {
		SettingsStatus.Log(logging.Error(), "Error opening file '"+settingsFile+"': "+err.Error())
		return nil, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil {
		SettingsStatus.Log(logging.Error(), "Error parsing json from file '"+settingsFile+"': "+err.Error())
		return nil, err
	}

	return data, nil
}

// writeFile to given file, TODO: create one if it doesn't exist
func writeFile(file string) error {
	settingsLock.Lock()
	settingsJSON, err := json.Marshal(Settings)
	settingsLock.Unlock()

	if err != nil {
		SettingsStatus.Log(logging.Error(), "Failed to marshall Settings")
		return err
	}

	err = ioutil.WriteFile(file, settingsJSON, 0644)
	if err != nil {
		SettingsStatus.Log(logging.Error(), "Failed to write Settings to "+file+": "+err.Error())
		return err
	}

	// Log success
	SettingsStatus.Log(logging.OK(), "Successfully wrote Settings to "+file)
	return nil
}

// HandleGetAll returns all current settings
func HandleGetAll(w http.ResponseWriter, r *http.Request) {
	if verboseOutput {
		SettingsStatus.Log(logging.OK(), "Responding to GET request with entire settings map.")
	}
	settingsLock.Lock()
	json.NewEncoder(w).Encode(Settings)
	settingsLock.Unlock()
}

// HandleGet returns all the values of a specific setting
func HandleGet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	componentName := formatting.FormatName(params["component"])

	if verboseOutput {
		SettingsStatus.Log(logging.OK(), fmt.Sprintf("Responding to GET request for setting component %s", componentName))
	}

	settingsLock.Lock()
	json.NewEncoder(w).Encode(Settings[componentName])
	settingsLock.Unlock()
}

// HandleGetValue returns a specific setting value
func HandleGetValue(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	componentName := formatting.FormatName(params["component"])
	settingName := formatting.FormatName(params["name"])

	if verboseOutput {
		SettingsStatus.Log(logging.OK(), fmt.Sprintf("Responding to GET request for setting %s on component %s", settingName, componentName))
	}

	json.NewEncoder(w).Encode(Settings[componentName][settingName])
}

// Get returns all the values of a specific setting
func Get(componentName string, settingName string) (string, error) {
	componentName = formatting.FormatName(componentName)

	if verboseOutput {
		SettingsStatus.Log(logging.OK(), fmt.Sprintf("Responding to request for setting component %s", componentName))
	}

	settingsLock.Lock()
	componenet, ok := Settings[componentName]
	if ok {
		setting, ok := componenet[settingName]
		if ok {
			settingsLock.Unlock()
			return setting, nil
		}
	}
	settingsLock.Unlock()
	return "", fmt.Errorf("Could not find componenet/setting with those values")
}

// HandleSet is the http wrapper for our setting setter
func HandleSet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	// Parse out params
	componentName := formatting.FormatName(params["component"])
	settingName := formatting.FormatName(params["name"])
	settingValue := params["value"]

	// Log if requested
	if verboseOutput {
		SettingsStatus.Log(logging.OK(), fmt.Sprintf("Responding to POST request for setting %s on component %s to be value %s", settingName, componentName, settingValue))
	}

	// Do the dirty work elsewhere
	Set(componentName, settingName, settingValue)

	// Respond with ack
	json.NewEncoder(w).Encode("OK")
}

// Set will handle actually updates or posts a new setting value
func Set(componentName string, settingName string, settingValue string) {
	// Insert componentName into Map if not exists
	settingsLock.Lock()
	if _, ok := Settings[componentName]; !ok {
		Settings[componentName] = make(map[string]string, 0)
	}

	// Update setting in inner map
	Settings[componentName][settingName] = settingValue
	settingsLock.Unlock()

	// Log our success
	SettingsStatus.Log(logging.OK(), fmt.Sprintf("Updated setting %s[%s] to %s", componentName, settingName, settingValue))

	// Write out all settings to a file
	writeFile(settingsFile)
}
