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
	zmq "github.com/pebbe/zmq4"
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
var sqlEnabled = false

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
	json.NewEncoder(w).Encode(Session[params["Name"]])
}

// UpdateSessionValue updates or posts a new session value to the common session
func UpdateSessionValue(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	var newdata SessionData
	_ = json.NewDecoder(r.Body).Decode(&newdata)

	// Set last updated time to now
	var timestamp = time.Now().Format("2006-01-02 15:04:05.999")
	newdata.LastUpdate = timestamp

	// Add / update value in global session
	Session[params["Name"]] = newdata

	// Check if MySQL logging has been turned off
	if sqlEnabled {
		// Insert into MySQL table
		_, err := DB.Exec("INSERT INTO log_session (timestamp, entry, value) values (?, ?, ?)", timestamp, params["Name"], newdata.Value)

		if err != nil {
			log.Println(err.Error())
			sqlEnabled = false
			DB.Close()
		} else {
			log.Println("Logged " + params["Name"] + " to sql db")
		}
	} else {
		log.Println("Recieved, but SQL logging not enabled")
	}

	// Respond with inserted values
	json.NewEncoder(w).Encode(newdata)
}

// SendPyBus queries a (hopefully) running pyBus program to run a directive
func SendPyBus(msg string) {
	context, _ := zmq.NewContext()
	socket, _ := context.NewSocket(zmq.REQ)
	defer socket.Close()

	log.Printf("Connecting to pyBus ZMQ Server")
	socket.Connect("tcp://localhost:4884")

	// Send command
	socket.Send(msg, 0)
	println("Sending PyBus command: ", msg)

	// Wait for reply:
	reply, _ := socket.Recv(0)
	println("Received: ", string(reply))
}

// PyBus handles gateway to PyBus goroutine
func PyBus(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	json.NewEncoder(w).Encode(Session[params["command"]])

	go SendPyBus(params["command"])
	json.NewEncoder(w).Encode(params["command"])
}
