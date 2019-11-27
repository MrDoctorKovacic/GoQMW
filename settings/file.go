package settings

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/rs/zerolog/log"
)

// ReadFile will handle the initialization of settings,
// either from past mapping or by creating a new one
func ReadFile(useSettingsFile string) {
	log.Info().Msg("Checking settings file...")
	if useSettingsFile == "" {
		log.Warn().Msg("Failed to load settings from file '" + Settings.File + "'. Is it empty?")
		return
	}

	Settings.File = useSettingsFile
	initSettings, err := parseFile(Settings.File)

	if err != nil || initSettings == nil || len(initSettings) == 0 {
		panic("Failed to load settings from file '" + Settings.File + "'. Is it empty?")
	}

	// Set new settings globally
	Settings.Data = initSettings

	// Run hooks on all new settings
	if out, err := json.Marshal(Settings.Data); err == nil {
		log.Info().Msg("Successfully loaded settings from file '" + Settings.File + "': " + string(out))
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
		log.Error().Msg("Error opening file '" + filename + "': " + err.Error())
		return nil, err
	}
	defer filep.Close()
	decoder := json.NewDecoder(filep)
	err = decoder.Decode(&data)
	if err != nil {
		log.Error().Msg("Error parsing json from file '" + filename + "': " + err.Error())
		return nil, err
	}

	return data, nil
}

// writeFile to given file, TODO: create one if it doesn't exist
func writeFile(file string) error {
	Settings.mutex.Lock()
	settingsJSON, err := json.MarshalIndent(Settings.Data, "", "\t")
	Settings.mutex.Unlock()

	if err != nil {
		log.Error().Msg("Failed to marshall Settings")
		return err
	}

	if err = ioutil.WriteFile(file, settingsJSON, 0644); err != nil {
		log.Error().Msg("Failed to write Settings to " + file + ": " + err.Error())
		return err
	}

	// Log success
	log.Info().Msg("Successfully wrote Settings to " + file)
	return nil
}
