// Package settings reads and writes to an MDroid settings file
package settings

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/influx"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/tarm/serial"

	"github.com/gorilla/mux"
)

// ConfigValues controls program settings and general persistent settings
type ConfigValues struct {
	BluetoothAddress      string
	DB                    *influx.Influx
	HardwareSerialEnabled bool
	HardwareSerialPort    string
	HardwareSerialBaud    string
	SerialControlDevice   *serial.Port
	SettingsFile          string
	SlackURL              string
}

// Settings control generic user defined field:value mappings, which will persist each run
var (
	Data         map[string]map[string]string // Main settings map
	Config       ConfigValues
	status       logging.ProgramStatus // status will control logging and reporting of status / warnings / errors
	settingsLock sync.Mutex            // Mutex allow concurrent reads/writes
)

func init() {
	status = logging.NewStatus("Settings")

	Config = ConfigValues{SettingsFile: "./settings.json"}

	// Default to empty map
	Data = make(map[string]map[string]string, 0)
}

// HandleGetAll returns all current settings
func HandleGetAll(w http.ResponseWriter, r *http.Request) {
	status.Log(logging.Debug(), "Responding to GET request with entire settings map.")

	settingsLock.Lock()
	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: Data, Status: "success", OK: true})
	settingsLock.Unlock()
}

// HandleGet returns all the values of a specific setting
func HandleGet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	componentName := formatting.FormatName(params["component"])

	status.Log(logging.Debug(), fmt.Sprintf("Responding to GET request for setting component %s", componentName))

	settingsLock.Lock()
	responseVal, ok := Data[componentName]
	settingsLock.Unlock()

	var response formatting.JSONResponse
	if !ok {
		response = formatting.JSONResponse{Output: "Setting not found.", Status: "fail", OK: false}
	} else {
		response = formatting.JSONResponse{Output: responseVal, Status: "success", OK: true}
	}

	json.NewEncoder(w).Encode(response)
}

// HandleGetValue returns a specific setting value
func HandleGetValue(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	componentName := formatting.FormatName(params["component"])
	settingName := formatting.FormatName(params["name"])

	status.Log(logging.Debug(), fmt.Sprintf("Responding to GET request for setting %s on component %s", settingName, componentName))

	settingsLock.Lock()
	responseVal, ok := Data[componentName][settingName]
	settingsLock.Unlock()

	var response formatting.JSONResponse
	if !ok {
		response = formatting.JSONResponse{Output: "Setting not found.", Status: "fail", OK: false}
	} else {
		response = formatting.JSONResponse{Output: responseVal, Status: "success", OK: true}
	}

	json.NewEncoder(w).Encode(response)
}

// GetAll returns all the values of known settings
func GetAll() map[string]map[string]string {
	status.Log(logging.Debug(), fmt.Sprintf("Responding to request for all settings"))

	settingsLock.Lock()
	settingsCopy := Data
	settingsLock.Unlock()
	return settingsCopy
}

// Get returns all the values of a specific setting
func Get(componentName string, settingName string) (string, error) {
	componentName = formatting.FormatName(componentName)
	status.Log(logging.Debug(), fmt.Sprintf("Responding to request for setting component %s", componentName))

	settingsLock.Lock()
	defer settingsLock.Unlock()
	component, ok := Data[componentName]
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
	status.Log(logging.Debug(), fmt.Sprintf("Responding to POST request for setting %s on component %s to be value %s", settingName, componentName, settingValue))

	// Do the dirty work elsewhere
	Set(componentName, settingName, settingValue)

	// Respond with OK
	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: componentName, Status: "success", OK: true})
}

// Set will handle actually updates or posts a new setting value
func Set(componentName string, settingName string, settingValue string) {
	// Insert componentName into Map if not exists
	settingsLock.Lock()
	if _, ok := Data[componentName]; !ok {
		Data[componentName] = make(map[string]string, 0)
	}

	// Update setting in inner map
	Data[componentName][settingName] = settingValue
	settingsLock.Unlock()

	// Log our success
	status.Log(logging.OK(), fmt.Sprintf("Updated setting %s[%s] to %s", componentName, settingName, settingValue))

	// Write out all settings to a file
	writeFile(Config.SettingsFile)
}
