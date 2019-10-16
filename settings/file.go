package settings

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/logging"
)

// ReadFile will handle the initialization of settings,
// either from past mapping or by creating a new one
func ReadFile(useSettingsFile string) {
	if useSettingsFile == "" {
		status.Log(logging.Warning(), "Failed to load settings from file '"+Settings.File+"'. Is it empty?")
		return
	}

	Settings.File = useSettingsFile
	initSettings, err := parseFile(Settings.File)
	defer Set("MDROID", "LAST_USED_UTC", time.Now().String())

	if err != nil || initSettings == nil || len(initSettings) == 0 {
		status.Log(logging.Warning(), "Failed to load settings from file '"+Settings.File+"'. Is it empty?")
		return
	}

	// Set new settings globally
	Settings.Data = initSettings

	// Run hooks on all new settings
	if out, err := json.Marshal(Settings.Data); err == nil {
		status.Log(logging.OK(), "Successfully loaded settings from file '"+Settings.File+"': "+string(out))
		for component := range Settings.Data {
			for setting := range Settings.Data[component] {
				runHooks(component, setting, Settings.Data[component][setting])
			}
		}
	}
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
	Settings.mutex.Lock()
	settingsJSON, err := json.Marshal(Settings.Data)
	Settings.mutex.Unlock()

	if err != nil {
		status.Log(logging.Error(), "Failed to marshall Settings")
		return err
	}

	if err = ioutil.WriteFile(file, settingsJSON, 0644); err != nil {
		status.Log(logging.Error(), "Failed to write Settings to "+file+": "+err.Error())
		return err
	}

	// Log success
	status.Log(logging.OK(), "Successfully wrote Settings to "+file)
	return nil
}
