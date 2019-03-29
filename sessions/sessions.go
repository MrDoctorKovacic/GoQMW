package sessions

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	// Import driver for MySQL
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

// SessionData holds the name, data, last update info for each session value
type SessionData struct {
	Value      string `json:"value,omitempty"`
	LastUpdate string `json:"lastUpdate,omitempty"`
}

// Session is the global session accessed by incoming requests
var Session map[string]SessionData

// DB MySQL variables
var DB *sql.DB

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

// SQLConnect to MySQL, provide global DB for future queries
func SQLConnect(database *sql.DB) {
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

	// Insert into MySQL table
	_, err := DB.Exec("INSERT INTO log_serial (timestamp, entry, value) values (?, ?, ?)", timestamp, params["name"], newdata.Value)

	if err != nil {
		log.Println(err.Error())
	} else {
		log.Println("Logged " + params["name"] + " to sql db")
	}

	// Respond with inserted values
	json.NewEncoder(w).Encode(newdata)
}

// LogALPR creates a new entry in running SQL DB
func LogALPR(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	// Decode plate/time/etc values
	plate := params["plate"]
	value := params["value"]

	// Insert into MySQL table
	// TODO: add location data
	_, err := DB.Exec("INSERT INTO log_alpr (timestamp, plate, percent) values (?, ?, ?)", time.Now().Format("2006-01-02 15:04:05.999"), plate, value)

	if err != nil {
		log.Println("Error executing SQL insert:")
		log.Println(err.Error())
	} else {
		log.Println("Logged " + plate + " to sql db")
	}

	// Respond with inserted values
	json.NewEncoder(w).Encode(plate)
}

// RestartALPR posts remote device to restart ALPR service
func RestartALPR(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("OK")
}
