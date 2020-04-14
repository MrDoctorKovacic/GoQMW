// Package settings reads and writes to an MDroid settings file
package settings

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/qcasey/MDroid-Core/format"
	"github.com/qcasey/MDroid-Core/format/response"
	"github.com/qcasey/MDroid-Core/mqtt"
	"github.com/rs/zerolog/log"

	"github.com/gorilla/mux"
)

type settingsWrap struct {
	File  string
	mutex sync.RWMutex
	Data  map[string]map[string]string // Main settings map
}

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

// Settings control generic user defined field:value mappings, which will persist each run
var Settings settingsWrap

func init() {
	Settings = settingsWrap{Data: make(map[string]map[string]string, 0)}
}

// HandleGetAll returns all current settings
func HandleGetAll(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msg("Responding to GET request with entire settings map.")
	resp := response.JSONResponse{Output: GetAll(), Status: "success", OK: true}
	resp.Write(&w, r)
}

// HandleGet returns all the values of a specific setting
func HandleGet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	componentName := format.Name(params["component"])

	log.Debug().Msgf("Responding to GET request for setting component %s", componentName)

	Settings.mutex.RLock()
	responseVal, ok := Settings.Data[componentName]
	Settings.mutex.RUnlock()

	resp := response.JSONResponse{Output: responseVal, OK: true}
	if !ok {
		resp = response.JSONResponse{Output: "Setting not found.", OK: false}
	}

	resp.Write(&w, r)
}

// HandleGetValue returns a specific setting value
func HandleGetValue(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	componentName := format.Name(params["component"])
	settingName := format.Name(params["name"])

	log.Debug().Msgf("Responding to GET request for setting %s on component %s", settingName, componentName)

	Settings.mutex.RLock()
	responseVal, ok := Settings.Data[componentName][settingName]
	Settings.mutex.RUnlock()

	resp := response.JSONResponse{Output: responseVal, OK: true}
	if !ok {
		resp = response.JSONResponse{Output: "Setting not found.", OK: false}
	}

	resp.Write(&w, r)
}

// GetAll returns all the values of known settings
func GetAll() map[string]map[string]string {
	log.Debug().Msgf("Responding to request for all settings")

	newData := map[string]map[string]string{}

	Settings.mutex.RLock()
	defer Settings.mutex.RUnlock()
	for index, element := range Settings.Data {
		newData[index] = element
	}

	return newData
}

// GetComponent returns all the values of a specific component
func GetComponent(componentName string) (map[string]string, error) {
	componentName = format.Name(componentName)
	log.Debug().Msgf("Responding to request for setting component %s", componentName)

	Settings.mutex.RLock()
	defer Settings.mutex.RUnlock()
	component, ok := Settings.Data[componentName]
	if ok {
		return component, nil
	}
	return nil, fmt.Errorf("Could not find component with name %s", componentName)
}

// Get returns all the values of a specific setting
func Get(componentName string, settingName string) (string, error) {
	Settings.mutex.RLock()
	defer Settings.mutex.RUnlock()

	component, ok := Settings.Data[format.Name(componentName)]
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
	componentName := format.Name(params["component"])
	settingName := format.Name(params["name"])
	settingValue := params["value"]

	// Log if requested
	log.Debug().Msgf("Responding to POST request for setting %s on component %s to be value %s", settingName, componentName, settingValue)

	// Do the dirty work elsewhere
	Set(componentName, settingName, settingValue)

	// Respond with OK
	response := response.JSONResponse{Output: componentName, OK: true}
	response.Write(&w, r)
}

// Set will handle actually updates or posts a new setting value
func Set(componentName string, settingName string, settingValue string) bool {
	// Format names
	componentName = format.Name(componentName)
	settingName = format.Name(settingName)
	settingValue = format.Name(settingValue)

	// Insert componentName into Map if not exists
	Settings.mutex.Lock()
	if _, ok := Settings.Data[componentName]; !ok {
		Settings.Data[componentName] = make(map[string]string, 0)
	}

	// Update setting in inner map
	Settings.Data[componentName][settingName] = settingValue
	Settings.mutex.Unlock()

	// Post to MQTT
	if mqtt.Enabled {
		topic := fmt.Sprintf("settings/%s/%s", componentName, settingName)
		go mqtt.Publish(topic, settingValue)
	}

	// Log our success
	log.Info().Msgf("Updated setting of %s[%s] to %s", componentName, settingName, settingValue)

	// Write out all settings to a file
	writeFile(Settings.File)

	// Trigger hooks
	runHooks(componentName, settingName, settingValue)

	return true
}
