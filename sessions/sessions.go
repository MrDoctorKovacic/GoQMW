package sessions

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/MrDoctorKovacic/GoQMW/influx"
	"github.com/gorilla/mux"
)

// SessionData holds the name, data, last update info for each session value
type SessionData struct {
	Value      string `json:"value,omitempty"`
	LastUpdate string `json:"lastUpdate,omitempty"`
}

// Session is the global session accessed by incoming requests
var Session map[string]SessionData

// DB variables
var DB influx.Influx

func init() {
	//
	// Fetch and append old session from disk
	//
	Session = make(map[string]SessionData)
	jsonFile, err := os.Open("session.json")

	if err != nil {
		log.Println("error opening JSON file on disk")
	} else {
		byteValue, _ := ioutil.ReadAll(jsonFile)
		json.Unmarshal(byteValue, &Session)
	}
}

// Setup provides global DB for future queries, other planned setup instructions as well
func Setup(database influx.Influx) {
	DB = database
}

// GetSession returns the entire current session
func GetSession(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(Session)
}

// GetSessionValue returns a specific session value
func GetSessionValue(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	json.NewEncoder(w).Encode(Session[params["name"]])
}

// SetSessionValue updates or posts a new session value to the common session
func SetSessionValue(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	var newdata SessionData
	_ = json.NewDecoder(r.Body).Decode(&newdata)

	// Set last updated time to now
	var timestamp = time.Now().Format("2006-01-02 15:04:05.999")
	newdata.LastUpdate = timestamp

	// Add / update value in global session
	Session[params["name"]] = newdata

	// Insert into database
	err := DB.Write(fmt.Sprintf("pybus,name=%s value=\"%s\"", params["name"], newdata.Value))

	if err != nil {
		log.Println(err.Error())
	} else {
		log.Println("[Session] Logged " + params["name"] + " to database")
	}

	// Respond with inserted values
	json.NewEncoder(w).Encode(newdata)
}

// LogALPR creates a new entry in running SQL DB
func LogALPR(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	// Decode plate/time/etc values
	plate := params["plate"]

	// Insert into database
	err := DB.Write(fmt.Sprintf("alpr,plate=%s percent=\"%s\"", plate, params["percent"]))

	if err != nil {
		log.Println(err.Error())
	} else {
		log.Println("[ALPR] Logged " + plate + " to database")
	}

	// Respond with inserted values
	json.NewEncoder(w).Encode(plate)
}

// RestartALPR posts remote device to restart ALPR service
func RestartALPR(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("OK")
}
