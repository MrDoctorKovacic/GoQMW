package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

//
// Is Influx logging a core aspect of the route? It's probably in here then.
//

// SessionData holds the data and last update info for each session value
type SessionData struct {
	Value      string `json:"value,omitempty"`
	LastUpdate string `json:"lastUpdate,omitempty"`
}

// SessionPackage contains both name and data
type SessionPackage struct {
	Name string
	Data SessionData
}

// Session is the global session accessed by incoming requests
var Session map[string]SessionData
var sessionLock sync.Mutex

// Session WebSocket upgrader
var upgrader = websocket.Upgrader{} // use default options

// SessionStatus will control logging and reporting of status / warnings / errors
var SessionStatus = logging.NewStatus("Session")

// SetupSessions will init the current session with a file
func SetupSessions(sessionFile string) {
	Session = make(map[string]SessionData)

	if sessionFile != "" {
		jsonFile, err := os.Open(sessionFile)

		if err != nil {
			SessionStatus.Log(logging.Warning(), "Error opening JSON file on disk: "+err.Error())
		} else {
			byteValue, _ := ioutil.ReadAll(jsonFile)
			json.Unmarshal(byteValue, &Session)
		}
	} else {
		SessionStatus.Log(logging.OK(), "Not saving or recovering from file")
	}

	// Setup triggers
	//initTriggers()
}

// HandleGetSession responds to an HTTP request for the entire session
func HandleGetSession(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(GetSession())
}

// GetSession returns the entire current session
func GetSession() map[string]SessionData {
	// Log if requested
	if Config.VerboseOutput {
		SessionStatus.Log(logging.OK(), "Responding to request for full session")
	}

	sessionLock.Lock()
	returnSession := Session
	sessionLock.Unlock()

	return returnSession
}

// GetSessionSocket returns the entire current session as a webstream
func GetSessionSocket(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true } // return true for now, although this should range over accepted origins

	// Log if requested
	if Config.VerboseOutput {
		SessionStatus.Log(logging.OK(), "Responding to request for session websocket")
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		SessionStatus.Log(logging.Error(), "Error upgrading webstream: "+err.Error())
		return
	}
	defer c.Close()
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			SessionStatus.Log(logging.Error(), "Error reading from webstream: "+err.Error())
			break
		}

		// Very verbose
		//SessionStatus.Log(logging.OK(), "Received: "+string(message))

		sessionLock.Lock()
		err = c.WriteJSON(Session)
		sessionLock.Unlock()

		if err != nil {
			SessionStatus.Log(logging.Error(), "Error writing to webstream: "+err.Error())
			break
		}
	}
}

// GetSessionValueHandler returns a specific session value
func GetSessionValueHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	sessionValue, err := GetSessionValue(params["name"])
	if err != nil {
		json.NewEncoder(w).Encode("Error: " + err.Error())
		return
	}

	json.NewEncoder(w).Encode(sessionValue)
}

// GetSessionValue returns the named session, if it exists. Nil otherwise
func GetSessionValue(name string) (value SessionData, err error) {

	// Log if requested
	if Config.VerboseOutput {
		SessionStatus.Log(logging.OK(), fmt.Sprintf("Responding to request for session value %s", name))
	}

	sessionLock.Lock()
	sessionValue, ok := Session[name]
	sessionLock.Unlock()

	if !ok {
		return sessionValue, fmt.Errorf("%s does not exist in Session", name)
	}

	return sessionValue, nil
}

// PostSessionValue updates or posts a new session value to the common session
func PostSessionValue(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	// Put body back
	r.Body.Close() //  must close
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	if len(body) == 0 {
		json.NewEncoder(w).Encode("Error: Empty body")
	}

	params := mux.Vars(r)
	var newdata SessionData
	err = json.NewDecoder(r.Body).Decode(&newdata)

	if err != nil {
		SessionStatus.Log(logging.Error(), "Error decoding incoming JSON")
		SessionStatus.Log(logging.Error(), err.Error())
		json.NewEncoder(w).Encode(err.Error())
		return
	}

	// Call the setter
	newPackage := SessionPackage{Name: params["name"], Data: newdata}
	err = newPackage.SetSessionValue(false)

	if err != nil {
		json.NewEncoder(w).Encode(err.Error())
		return
	}

	// Respond with success
	json.NewEncoder(w).Encode("OK")
}

// SetSessionValue does the actual setting of Session Values
func (newPackage *SessionPackage) SetSessionValue(quiet bool) error {
	// Ensure name is valid
	if !formatting.IsValidName(newPackage.Name) {
		return fmt.Errorf("%s is not a valid name. Possibly a failed serial transmission?", newPackage.Name)
	}

	// Set last updated time to now
	var timestamp = time.Now().In(Timezone).Format("2006-01-02 15:04:05.999")
	newPackage.Data.LastUpdate = timestamp

	// Correct name
	newPackage.Name = formatting.FormatName(newPackage.Name)

	// Trim off whitespace
	newPackage.Data.Value = strings.TrimSpace(newPackage.Data.Value)

	// Log if requested
	if Config.VerboseOutput && !quiet {
		SessionStatus.Log(logging.OK(), fmt.Sprintf("Responding to request for session key %s = %s", newPackage.Name, newPackage.Data.Value))
	}

	// Add / update value in global session after locking access to session
	sessionLock.Lock()
	Session[newPackage.Name] = newPackage.Data
	sessionLock.Unlock()

	// Finish post processing
	go newPackage.processSessionTriggers()

	// Insert into database
	if Config.DatabaseEnabled {

		// Convert to a float if that suits the value, otherwise change field to value_string
		var valueString string
		if _, err := strconv.ParseFloat(newPackage.Data.Value, 32); err == nil {
			valueString = fmt.Sprintf("value=%s", newPackage.Data.Value)
		} else {
			valueString = fmt.Sprintf("value_string=\"%s\"", newPackage.Data.Value)
		}

		// In Sessions, all values come in and out as strings regardless,
		// but this conversion alows Influx queries on the floats to be executed
		err := Config.DB.Write(fmt.Sprintf("pybus,name=%s %s", strings.Replace(newPackage.Name, " ", "_", -1), valueString))

		if err != nil {
			errorText := fmt.Sprintf("Error writing %s=%s to influx DB: %s", newPackage.Name, newPackage.Data.Value, err.Error())
			SessionStatus.Log(logging.Error(), errorText)
			return errors.New(errorText)
		} else if !quiet {
			SessionStatus.Log(logging.OK(), fmt.Sprintf("Logged %s=%s to database", newPackage.Name, newPackage.Data.Value))
		}
	}

	return nil
}

// SetSessionRawValue prepares a SessionData structure before passing it to the setter
func SetSessionRawValue(name string, value string) {
	newPackage := SessionPackage{Name: name, Data: SessionData{Value: value}}
	newPackage.SetSessionValue(true)
}
