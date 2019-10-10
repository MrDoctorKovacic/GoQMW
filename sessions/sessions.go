package sessions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/gps"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/gorilla/mux"
)

// Value holds the data and last update info for each session value
type Value struct {
	Value      string `json:"value,omitempty"`
	LastUpdate string `json:"lastUpdate,omitempty"`
	Quiet      bool   `json:"quiet,omitempty"`
}

// sessionPackage contains both name and data
type sessionPackage struct {
	Name string
	Data Value
}

// Session is a mapping of sessionPackages, which contain session values
type Session struct {
	data  map[string]Value
	Mutex sync.Mutex
	file  string
}

var (
	status  logging.ProgramStatus
	session Session
)

func init() {
	// status will control logging and reporting of status / warnings / errors
	status = logging.NewStatus("Session")
}

// Create will init the current session with a file
func Create(sessionFile string) {
	session.data = make(map[string]Value)

	if sessionFile != "" {
		session.file = sessionFile
		jsonFile, err := os.Open(sessionFile)

		if err != nil {
			status.Log(logging.Warning(), "Error opening JSON file on disk: "+err.Error())
		} else {
			byteValue, err := ioutil.ReadAll(jsonFile)
			if err != nil {
				status.Log(logging.Error(), err.Error())
				return
			}
			json.Unmarshal(byteValue, &session)
		}
	} else {
		status.Log(logging.OK(), "Not saving or recovering from file")
	}
}

// HandleGetAll responds to an HTTP request for the entire session
func HandleGetAll(w http.ResponseWriter, r *http.Request) {
	fullSession := GetAll()
	response := formatting.JSONResponse{Output: fullSession, Status: "success", OK: true}
	json.NewEncoder(w).Encode(response)
}

// GetAll returns the entire current session
func GetAll() map[string]Value {
	// Log if requested
	status.Log(logging.Debug(), "Responding to request for full session")

	session.Mutex.Lock()
	returnSession := session.data
	session.Mutex.Unlock()

	return returnSession
}

// HandleGet returns a specific session value
func HandleGet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	sessionValue, err := Get(params["name"])
	response := formatting.JSONResponse{}
	if err != nil {
		response.Status = "fail"
		response.Output = err.Error()
		response.OK = false
		json.NewEncoder(w).Encode(response)
		return
	}

	// Craft OK response
	response.Status = "success"
	response.Output = sessionValue
	response.OK = true

	json.NewEncoder(w).Encode(response)
}

// Get returns the named session, if it exists. Nil otherwise
func Get(name string) (value Value, err error) {

	// Log if requested
	status.Log(logging.Debug(), fmt.Sprintf("Responding to request for session value %s", name))

	session.Mutex.Lock()
	sessionValue, ok := session.data[name]
	session.Mutex.Unlock()

	if !ok {
		return sessionValue, fmt.Errorf("%s does not exist in Session", name)
	}

	return sessionValue, nil
}

// GetBool returns the named session with a boolean value, if it exists. false otherwise
func GetBool(name string) (value bool, err error) {
	v, err := Get(name)
	if err != nil {
		return false, err
	}

	vb, err := strconv.ParseBool(v.Value)
	if err != nil {
		return false, err
	}
	return vb, nil
}

// HandlePost updates or posts a new session value to the common session
func HandlePost(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)

	// Default to NOT OK response
	response := formatting.JSONResponse{Status: "fail", OK: false}

	if err != nil {
		status.Log(logging.Error(), fmt.Sprintf("Error reading body: %v", err))
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	// Put body back
	r.Body.Close() //  must close
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	if len(body) == 0 {
		response.Output = "Error: Empty body"
		json.NewEncoder(w).Encode(response)
	}

	params := mux.Vars(r)
	var newdata Value
	err = json.NewDecoder(r.Body).Decode(&newdata)

	if err != nil {
		status.Log(logging.Error(), fmt.Sprintf("Error decoding incoming JSON:\n%s", err.Error()))

		response.Output = err.Error()
		json.NewEncoder(w).Encode(response)
		return
	}

	// Call the setter
	newPackage := sessionPackage{Name: params["name"], Data: newdata}
	err = Set(newPackage, newdata.Quiet)

	if err != nil {
		response.Output = err.Error()
		json.NewEncoder(w).Encode(response)
		return
	}

	// Craft OK response
	response.Status = "success"
	response.OK = true
	response.Output = newPackage

	// Respond with success
	json.NewEncoder(w).Encode(response)
}

// SetValue prepares a Value structure before passing it to the setter
func SetValue(name string, value string) {
	Set(sessionPackage{Name: name, Data: Value{Value: value}}, true)
}

// Set does the actual setting of Session Values
func Set(newPackage sessionPackage, quiet bool) error {
	// Ensure name is valid
	if !formatting.IsValidName(newPackage.Name) {
		return fmt.Errorf("%s is not a valid name. Possibly a failed serial transmission?", newPackage.Name)
	}

	// Set last updated time to now
	var timestamp = time.Now().In(gps.GetTimezone()).Format("2006-01-02 15:04:05.999")
	newPackage.Data.LastUpdate = timestamp

	// Correct name
	newPackage.Name = formatting.FormatName(newPackage.Name)

	// Trim off whitespace
	newPackage.Data.Value = strings.TrimSpace(newPackage.Data.Value)

	// Log if requested
	if !quiet {
		status.Log(logging.Debug(), fmt.Sprintf("Responding to request for session key %s = %s", newPackage.Name, newPackage.Data.Value))
	}

	// Add / update value in global session after locking access to session
	session.Mutex.Lock()
	session.data[newPackage.Name] = newPackage.Data
	session.Mutex.Unlock()

	// Finish post processing
	go processSessionTriggers(newPackage)

	// Insert into database
	if settings.Config.DB != nil {

		// Convert to a float if that suits the value, otherwise change field to value_string
		var valueString string
		if _, err := strconv.ParseFloat(newPackage.Data.Value, 32); err == nil {
			valueString = fmt.Sprintf("value=%s", newPackage.Data.Value)
		} else {
			valueString = fmt.Sprintf("value_string=\"%s\"", newPackage.Data.Value)
		}

		// In Sessions, all values come in and out as strings regardless,
		// but this conversion alows Influx queries on the floats to be executed
		online, err := settings.Config.DB.Write(fmt.Sprintf("pybus,name=%s %s", strings.Replace(newPackage.Name, " ", "_", -1), valueString))

		if err != nil {
			errorText := fmt.Sprintf("Error writing %s=%s to influx DB: %s", newPackage.Name, newPackage.Data.Value, err.Error())
			// Only spam our log if Influx is online
			if online {
				status.Log(logging.Error(), errorText)
			}
			return fmt.Errorf(errorText)
		} else if !quiet {
			status.Log(logging.OK(), fmt.Sprintf("Logged %s=%s to database", newPackage.Name, newPackage.Data.Value))
		}
	}

	return nil
}
