// Package settings reads and writes to an MDroid settings file
package settings

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/rs/zerolog/log"
	"github.com/tarm/serial"

	"github.com/gorilla/mux"
)

type settingsWrap struct {
	File  string
	mutex sync.Mutex
	Data  map[string]map[string]string // Main settings map
}

// Settings control generic user defined field:value mappings, which will persist each run
var (
	Settings            settingsWrap
	BluetoothAddress    string
	SerialControlDevice *serial.Port
	SlackURL            string
)

func init() {
	Settings = settingsWrap{File: "./settings.json", Data: make(map[string]map[string]string, 0)}
}

// HandleGetAll returns all current settings
func HandleGetAll(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msg("Responding to GET request with entire settings map.")
	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: GetAll(), Status: "success", OK: true})
}

// HandleGet returns all the values of a specific setting
func HandleGet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	componentName := formatting.FormatName(params["component"])

	log.Debug().Msg(fmt.Sprintf("Responding to GET request for setting component %s", componentName))

	Settings.mutex.Lock()
	responseVal, ok := Settings.Data[componentName]
	Settings.mutex.Unlock()

	response := formatting.JSONResponse{Output: responseVal, Status: "success", OK: true}
	if !ok {
		response = formatting.JSONResponse{Output: "Setting not found.", Status: "fail", OK: false}
	}

	json.NewEncoder(w).Encode(response)
}

// HandleGetValue returns a specific setting value
func HandleGetValue(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	componentName := formatting.FormatName(params["component"])
	settingName := formatting.FormatName(params["name"])

	log.Debug().Msg(fmt.Sprintf("Responding to GET request for setting %s on component %s", settingName, componentName))

	Settings.mutex.Lock()
	responseVal, ok := Settings.Data[componentName][settingName]
	Settings.mutex.Unlock()

	response := formatting.JSONResponse{Output: responseVal, Status: "success", OK: true}
	if !ok {
		response = formatting.JSONResponse{Output: "Setting not found.", Status: "fail", OK: false}
	}

	json.NewEncoder(w).Encode(response)
}

// GetAll returns all the values of known settings
func GetAll() map[string]map[string]string {
	log.Debug().Msg(fmt.Sprintf("Responding to request for all settings"))

	newData := map[string]map[string]string{}

	Settings.mutex.Lock()
	defer Settings.mutex.Unlock()
	for index, element := range Settings.Data {
		newData[index] = element
	}

	return newData
}

// GetComponent returns all the values of a specific component
func GetComponent(componentName string) (map[string]string, error) {
	componentName = formatting.FormatName(componentName)
	log.Debug().Msg(fmt.Sprintf("Responding to request for setting component %s", componentName))

	Settings.mutex.Lock()
	defer Settings.mutex.Unlock()
	component, ok := Settings.Data[componentName]
	if ok {
		return component, nil
	}
	return nil, fmt.Errorf("Could not find component/setting with those values")
}

// Get returns all the values of a specific setting
func Get(componentName string, settingName string) (string, error) {
	componentName = formatting.FormatName(componentName)
	log.Debug().Msg(fmt.Sprintf("Responding to request for setting component %s", componentName))

	Settings.mutex.Lock()
	defer Settings.mutex.Unlock()
	component, ok := Settings.Data[componentName]
	if ok {
		setting, ok := component[settingName]
		if ok {
			return setting, nil
		}
	}
	return "", fmt.Errorf("Could not find component/setting with those values")
}

// GetBool returns the named session with a boolean value, if it exists. false otherwise
func GetBool(componentName string, settingName string) (value bool, err error) {
	v, err := Get(componentName, settingName)
	if err != nil {
		return false, err
	}

	vb, err := strconv.ParseBool(v)
	if err != nil {
		return false, err
	}
	return vb, nil
}

// HandleSet is the http wrapper for our setting setter
func HandleSet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	// Parse out params
	componentName := formatting.FormatName(params["component"])
	settingName := formatting.FormatName(params["name"])
	settingValue := params["value"]

	// Log if requested
	log.Debug().Msg(fmt.Sprintf("Responding to POST request for setting %s on component %s to be value %s", settingName, componentName, settingValue))

	// Do the dirty work elsewhere
	Set(componentName, settingName, settingValue)

	// Respond with OK
	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: componentName, Status: "success", OK: true})
}

// Set will handle actually updates or posts a new setting value
func Set(componentName string, settingName string, settingValue string) {
	// Insert componentName into Map if not exists
	Settings.mutex.Lock()
	if _, ok := Settings.Data[componentName]; !ok {
		Settings.Data[componentName] = make(map[string]string, 0)
	}

	// Update setting in inner map
	Settings.Data[componentName][settingName] = settingValue
	Settings.mutex.Unlock()

	// Log our success
	log.Info().Msg(fmt.Sprintf("Updated setting %s[%s] to %s", componentName, settingName, settingValue))

	// Write out all settings to a file
	writeFile(Settings.File)

	// Trigger hooks
	runHooks(componentName, settingName, settingValue)
}
