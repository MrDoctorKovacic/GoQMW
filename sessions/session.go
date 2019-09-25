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
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
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

type Session struct {
	data   map[string]Value
	Mutex  sync.Mutex
	Config *settings.ConfigValues
}

// Session WebSocket upgrader
var upgrader = websocket.Upgrader{} // use default options

// SessionStatus will control logging and reporting of status / warnings / errors
var SessionStatus = logging.NewStatus("Session")

// CreateSession will init the current session with a file
func CreateSession(sessionFile string) Session {
	var session Session
	session.data = make(map[string]Value)

	if sessionFile != "" {
		jsonFile, err := os.Open(sessionFile)

		if err != nil {
			SessionStatus.Log(logging.Warning(), "Error opening JSON file on disk: "+err.Error())
		} else {
			byteValue, _ := ioutil.ReadAll(jsonFile)
			json.Unmarshal(byteValue, &session)
		}
	} else {
		SessionStatus.Log(logging.OK(), "Not saving or recovering from file")
	}
	return session
}

// HandleGetSession responds to an HTTP request for the entire session
func (session *Session) HandleGetSession(w http.ResponseWriter, r *http.Request) {
	response := formatting.JSONResponse{Output: session.GetSession(), Status: "success", OK: true}
	json.NewEncoder(w).Encode(response)
}

// GetSession returns the entire current session
func (session *Session) GetSession() map[string]Value {
	// Log if requested
	if session.Config.VerboseOutput {
		SessionStatus.Log(logging.OK(), "Responding to request for full session")
	}

	session.Mutex.Lock()
	returnSession := session.data
	session.Mutex.Unlock()

	return returnSession
}

// GetSessionSocket returns the entire current session as a webstream
func (session *Session) GetSessionSocket(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true } // return true for now, although this should range over accepted origins

	// Log if requested
	if session.Config.VerboseOutput {
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

		// Pass through lock first
		writeSession := session.GetSession()

		err = c.WriteJSON(writeSession)

		if err != nil {
			SessionStatus.Log(logging.Error(), "Error writing to webstream: "+err.Error())
			break
		}
	}
}

// HandleGetSessionValue returns a specific session value
func (session *Session) HandleGetSessionValue(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	sessionValue, err := session.GetSessionValue(params["name"])
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

// GetSessionValue returns the named session, if it exists. Nil otherwise
func (session *Session) GetSessionValue(name string) (value Value, err error) {

	// Log if requested
	if session.Config.VerboseOutput {
		SessionStatus.Log(logging.OK(), fmt.Sprintf("Responding to request for session value %s", name))
	}

	session.Mutex.Lock()
	sessionValue, ok := session.data[name]
	session.Mutex.Unlock()

	if !ok {
		return sessionValue, fmt.Errorf("%s does not exist in Session", name)
	}

	return sessionValue, nil
}

// HandlePostSessionValue updates or posts a new session value to the common session
func (session *Session) HandlePostSessionValue(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)

	// Default to NOT OK response
	response := formatting.JSONResponse{}
	response.Status = "fail"
	response.OK = false

	if err != nil {
		SessionStatus.Log(logging.Error(), fmt.Sprintf("Error reading body: %v", err))
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
		SessionStatus.Log(logging.Error(), "Error decoding incoming JSON")
		SessionStatus.Log(logging.Error(), err.Error())

		response.Output = err.Error()
		json.NewEncoder(w).Encode(response)
		return
	}

	// Call the setter
	newPackage := sessionPackage{Name: params["name"], Data: newdata}
	err = session.SetSessionValue(newPackage, newdata.Quiet)

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

// CreateSessionValue prepares a Value structure before passing it to the setter
func (session *Session) CreateSessionValue(name string, value string) {
	session.SetSessionValue(sessionPackage{Name: name, Data: Value{Value: value}}, true)
}

// SetSessionValue does the actual setting of Session Values
func (session *Session) SetSessionValue(newPackage sessionPackage, quiet bool) error {
	// Ensure name is valid
	if !formatting.IsValidName(newPackage.Name) {
		return fmt.Errorf("%s is not a valid name. Possibly a failed serial transmission?", newPackage.Name)
	}

	// Set last updated time to now
	var timestamp = time.Now().In(session.Config.Timezone).Format("2006-01-02 15:04:05.999")
	newPackage.Data.LastUpdate = timestamp

	// Correct name
	newPackage.Name = formatting.FormatName(newPackage.Name)

	// Trim off whitespace
	newPackage.Data.Value = strings.TrimSpace(newPackage.Data.Value)

	// Log if requested
	if session.Config.VerboseOutput && !quiet {
		SessionStatus.Log(logging.OK(), fmt.Sprintf("Responding to request for session key %s = %s", newPackage.Name, newPackage.Data.Value))
	}

	// Add / update value in global session after locking access to session
	session.Mutex.Lock()
	session.data[newPackage.Name] = newPackage.Data
	session.Mutex.Unlock()

	// Finish post processing
	go session.processSessionTriggers(&newPackage)

	// Insert into database
	if session.Config.DB != nil {

		// Convert to a float if that suits the value, otherwise change field to value_string
		var valueString string
		if _, err := strconv.ParseFloat(newPackage.Data.Value, 32); err == nil {
			valueString = fmt.Sprintf("value=%s", newPackage.Data.Value)
		} else {
			valueString = fmt.Sprintf("value_string=\"%s\"", newPackage.Data.Value)
		}

		// In Sessions, all values come in and out as strings regardless,
		// but this conversion alows Influx queries on the floats to be executed
		online, err := session.Config.DB.Write(fmt.Sprintf("pybus,name=%s %s", strings.Replace(newPackage.Name, " ", "_", -1), valueString))

		if err != nil {
			errorText := fmt.Sprintf("Error writing %s=%s to influx DB: %s", newPackage.Name, newPackage.Data.Value, err.Error())
			// Only spam our log if Influx is online
			if online {
				SessionStatus.Log(logging.Error(), errorText)
			}
			return fmt.Errorf(errorText)
		} else if !quiet {
			SessionStatus.Log(logging.OK(), fmt.Sprintf("Logged %s=%s to database", newPackage.Name, newPackage.Data.Value))
		}
	}

	return nil
}
