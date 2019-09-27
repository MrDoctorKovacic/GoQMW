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
func ReadFile(useSettingsFile string) {
	if useSettingsFile == "" {
		status.Log(logging.Warning(), "Failed to load settings from file '"+Config.SettingsFile+"'. Is it empty?")
		return
	}

	Config.SettingsFile = useSettingsFile
	initSettings, err := parseFile(Config.SettingsFile)
	if err == nil && initSettings != nil && len(initSettings) != 0 {
		Data = initSettings

		// Check if we're configed to verbose output
		var verboseOutputInt int
		useVerboseOutput, ok := Data["MDROID"]["VERBOSE_OUTPUT"]
		if !ok {
			verboseOutputInt = 0
		} else {
			verboseOutputInt, err = strconv.Atoi(useVerboseOutput)
			if err != nil {
				verboseOutputInt = 0
			}
		}

		// Set as bool for return
		Config.VerboseOutput = verboseOutputInt != 0

		// Log settings
		out, err := json.Marshal(Data)
		if err == nil {
			status.Log(logging.OK(), "Successfully loaded settings from file '"+Config.SettingsFile+"': "+string(out))
			return
		}

		// If err is set, re-marshaling the settings failed
		status.Log(logging.Warning(), "Failed to load settings from file '"+Config.SettingsFile+"'. Defaulting to empty Map. Error: "+err.Error())
	} else if initSettings == nil {
		status.Log(logging.Warning(), "Failed to load settings from file '"+Config.SettingsFile+"'. Is it empty?")
	}

	Set("MDROID", "LAST_USED", time.Now().String())

	// Return empty map
	return
}

// parseFile will open and interpret program settings,
// as well as return the generic settings from last session
func parseFile(filename string) (map[string]map[string]string, error) {
	var data map[string]map[string]string

	// Open settings file
	filep, err := os.Open(filename)
	if err != nil {
		status.Log(logging.Error(), "Error opening file '"+filename+"': "+err.Error())
		return nil, err
	}
	defer filep.Close()
	decoder := json.NewDecoder(filep)
	err = decoder.Decode(&data)
	if err != nil {
		status.Log(logging.Error(), "Error parsing json from file '"+filename+"': "+err.Error())
		return nil, err
	}

	return data, nil
}

// writeFile to given file, TODO: create one if it doesn't exist
func writeFile(file string) error {
	settingsLock.Lock()
	settingsJSON, err := json.Marshal(Data)
	settingsLock.Unlock()

	if err != nil {
		status.Log(logging.Error(), "Failed to marshall Settings")
		return err
	}

	err = ioutil.WriteFile(file, settingsJSON, 0644)
	if err != nil {
		status.Log(logging.Error(), "Failed to write Settings to "+file+": "+err.Error())
		return err
	}

	// Log success
	status.Log(logging.OK(), "Successfully wrote Settings to "+file)
	return nil
}
