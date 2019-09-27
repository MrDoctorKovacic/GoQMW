package settings

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/logging"
)

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

			// Check if we're configed to verbose output
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
