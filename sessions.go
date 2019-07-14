package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/status"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

//
// Is Influx logging a core aspect of the route? It's probably in here then.
//

// SessionData holds the name, data, last update info for each session value
type SessionData struct {
	Value      string `json:"value,omitempty"`
	LastUpdate string `json:"lastUpdate,omitempty"`
}

// GPSData holds various data points we expect to receive
type GPSData struct {
	Latitude  string   `json:"latitude,omitempty"`
	Longitude string   `json:"longitude,omitempty"`
	Time      string   `json:"time,omitempty"` // This will help measure latency :)
	Altitude  *float32 `json:"altitude,omitempty"`
	EPV       *float32 `json:"epv,omitempty"`
	EPT       *float32 `json:"ept,omitempty"`
	Speed     *float32 `json:"speed,omitempty"`
	Climb     *float32 `json:"climb,omitempty"`
}

// ALPRData holds the plate and percent for each new ALPR value
type ALPRData struct {
	Plate   string `json:"plate,omitempty"`
	Percent int    `json:"percent,omitempty"`
}

// Session is the global session accessed by incoming requests
var Session map[string]SessionData
var sessionLock sync.Mutex

// GPS is the last posted GPS fix
var GPS GPSData

// Session WebSocket upgrader
var upgrader = websocket.Upgrader{} // use default options

// SessionStatus will control logging and reporting of status / warnings / errors
var SessionStatus = status.NewStatus("Session")

// Init session
func SetupSessions(sessionFile string) {
	Session = make(map[string]SessionData)

	if sessionFile != "" {
		jsonFile, err := os.Open(sessionFile)

		if err != nil {
			SessionStatus.Log(status.Warning(), "Error opening JSON file on disk: "+err.Error())
		} else {
			byteValue, _ := ioutil.ReadAll(jsonFile)
			json.Unmarshal(byteValue, &Session)
		}
	} else {
		SessionStatus.Log(status.OK(), "Not saving or recovering from file")
	}
}

// GetSession returns the entire current session
func GetSession(w http.ResponseWriter, r *http.Request) {
	// Log if requested
	if VERBOSE_OUTPUT {
		SessionStatus.Log(status.OK(), "Responding to GET request for full session")
	}

	sessionLock.Lock()
	json.NewEncoder(w).Encode(Session)
	sessionLock.Unlock()
}

// GetSessionSocket returns the entire current session as a webstream
func GetSessionSocket(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true } // return true for now, although this should range over accepted origins

	// Log if requested
	if VERBOSE_OUTPUT {
		SessionStatus.Log(status.OK(), "Responding to request for session websocket")
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		SessionStatus.Log(status.Error(), "Error upgrading webstream: "+err.Error())
		return
	}
	defer c.Close()
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			SessionStatus.Log(status.Error(), "Error reading from webstream: "+err.Error())
			break
		}

		// Very verbose
		//SessionStatus.Log(status.OK(), "Received: "+string(message))

		sessionLock.Lock()
		err = c.WriteJSON(Session)
		sessionLock.Unlock()

		if err != nil {
			SessionStatus.Log(status.Error(), "Error writing to webstream: "+err.Error())
			break
		}
	}
}

// GetSessionValue returns a specific session value
func GetSessionValue(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	// Log if requested
	if VERBOSE_OUTPUT {
		SessionStatus.Log(status.OK(), fmt.Sprintf("Responding to GET request for session value %s", params["name"]))
	}

	sessionLock.Lock()
	json.NewEncoder(w).Encode(Session[params["name"]])
	sessionLock.Unlock()
}

// SetSessionValue updates or posts a new session value to the common session
func SetSessionValue(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	// Put body back
	r.Body.Close() //  must close
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	if len(body) > 0 {

		params := mux.Vars(r)
		var newdata SessionData
		err = json.NewDecoder(r.Body).Decode(&newdata)

		if err != nil {
			SessionStatus.Log(status.Error(), "Error decoding incoming JSON")
			SessionStatus.Log(status.Error(), err.Error())
		}

		// Set last updated time to now
		var timestamp = time.Now().In(TIMEZONE).Format("2006-01-02 15:04:05.999")
		newdata.LastUpdate = timestamp

		// Trim off whitespace
		newdata.Value = strings.TrimSpace(newdata.Value)

		// Log if requested
		if VERBOSE_OUTPUT {
			SessionStatus.Log(status.OK(), fmt.Sprintf("Responding to POST request for session key %s = %s", params["name"], newdata.Value))
		}

		// Lock access to session
		sessionLock.Lock()

		// Add / update value in global session
		Session[params["name"]] = newdata

		// Immediately unlock, since defer could take a while
		sessionLock.Unlock()

		// Insert into database
		if DATABASE_ENABLED {

			// Convert to a float if that suits the value, otherwise change field to value_string
			var valueString string
			if _, err := strconv.ParseFloat(newdata.Value, 32); err == nil {
				valueString = fmt.Sprintf("value=%s", newdata.Value)
			} else {
				valueString = fmt.Sprintf("value_string=\"%s\"", newdata.Value)
			}

			// In Sessions, all values come in and out as strings regardless,
			// but this conversion alows Influx queries on the floats to be executed
			err := DB.Write(fmt.Sprintf("pybus,name=%s %s", strings.Replace(params["name"], " ", "_", -1), valueString))

			if err != nil {
				SessionStatus.Log(status.Error(), "Error writing "+params["name"]+"="+newdata.Value+" to influx DB: "+err.Error())
			} else {
				SessionStatus.Log(status.OK(), "Logged "+params["name"]+"="+newdata.Value+" to database")
			}
		}

		json.NewEncoder(w).Encode("OK")

	} else {
		json.NewEncoder(w).Encode("FAIL")
	}
}

//
// GPS Functions
//

// GetGPSValue returns the latest GPS fix
func GetGPSValue(w http.ResponseWriter, r *http.Request) {
	// Log if requested
	if VERBOSE_OUTPUT {
		SessionStatus.Log(status.OK(), "Responding to GET request for all GPS values")
	}
	json.NewEncoder(w).Encode(GPS)
}

// SetGPSValue posts a new GPS fix
func SetGPSValue(w http.ResponseWriter, r *http.Request) {
	var newdata GPSData
	_ = json.NewDecoder(r.Body).Decode(&newdata)

	// Log if requested
	if VERBOSE_OUTPUT {
		SessionStatus.Log(status.OK(), "Responding to POST request for gps values")
	}

	// Prepare new value
	var postingString strings.Builder

	// Update value for global session if the data is newer (not nil)
	// Can't find a better way to go about this
	if newdata.Latitude != "" {
		GPS.Latitude = newdata.Latitude
		postingString.WriteString(fmt.Sprintf("latitude=\"%s\",", newdata.Latitude))
	}
	if newdata.Longitude != "" {
		GPS.Longitude = newdata.Longitude
		postingString.WriteString(fmt.Sprintf("longitude=\"%s\",", newdata.Longitude))
	}
	if newdata.Altitude != nil {
		GPS.Altitude = newdata.Altitude
		log.Println(fmt.Sprintf("%f", *newdata.Altitude))
		postingString.WriteString(fmt.Sprintf("altitude=%f,", *newdata.Altitude))
	}
	if newdata.Speed != nil {
		GPS.Speed = newdata.Speed
		postingString.WriteString(fmt.Sprintf("speed=%f,", *newdata.Speed))
	}
	if newdata.Climb != nil {
		GPS.Climb = newdata.Climb
		postingString.WriteString(fmt.Sprintf("climb=%f,", *newdata.Climb))
	}
	if newdata.Time != "" {
		GPS.Time = newdata.Time
	}
	if newdata.EPV != nil {
		GPS.EPV = newdata.EPV
		postingString.WriteString(fmt.Sprintf("EPV=%f,", *newdata.EPV))
	}
	if newdata.EPT != nil {
		GPS.EPT = newdata.EPT
		postingString.WriteString(fmt.Sprintf("EPT=%f,", *newdata.EPT))
	}

	// Insert into database
	if DATABASE_ENABLED {
		err := DB.Write(fmt.Sprintf("gps %s", strings.TrimSuffix(postingString.String(), ",")))

		if err != nil {
			SessionStatus.Log(status.Error(), "Error writing string "+postingString.String()+" to influx DB: "+err.Error())
		} else {
			SessionStatus.Log(status.OK(), "Logged "+postingString.String()+" to database")
		}
	}

	json.NewEncoder(w).Encode("OK")
}

//
// ALPR Functions
//

// LogALPR creates a new entry in running SQL DB
func LogALPR(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	decoder := json.NewDecoder(r.Body)
	var newplate ALPRData
	err := decoder.Decode(&newplate)

	// Log if requested
	if VERBOSE_OUTPUT {
		SessionStatus.Log(status.OK(), "Responding to POST request for ALPR")
	}

	if err != nil {
		SessionStatus.Log(status.Error(), "Error decoding incoming ALPR data: "+err.Error())
	} else {
		// Decode plate/time/etc values
		plate := strings.Replace(params["plate"], " ", "_", -1)
		percent := newplate.Percent

		if plate != "" {
			// Insert into database
			if DATABASE_ENABLED {
				err := DB.Write(fmt.Sprintf("alpr,plate=%s percent=%d", plate, percent))

				if err != nil {
					SessionStatus.Log(status.Error(), "Error writing "+plate+" to influx DB: "+err.Error())
				} else {
					SessionStatus.Log(status.OK(), "Logged "+plate+" to database")
				}
			}
		} else {
			SessionStatus.Log(status.Error(), fmt.Sprintf("Missing arguments, ignoring post of %s with percent of %d", plate, percent))
		}
	}

	json.NewEncoder(w).Encode("OK")
}

// RestartALPR posts remote device to restart ALPR service
func RestartALPR(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("OK")
}