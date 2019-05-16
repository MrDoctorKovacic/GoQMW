package status

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Status of a particular program
// Keep some logs in memory
type Status struct {
	Name       string // should match the name in the StatusMap
	Status     string
	LastUpdate string
	DebugLog   []string // For OK messages, or general purpose logging
	WarningLog []string
	ErrorLog   []string
}

// MessageType for implementing all types of messages listed above
type MessageType struct {
	Name    string `json:"type,omitempty"`
	Message string `json:"message,omitempty"` // Only used for external logging, since they have the struct by name
}

// Error for logging critial faults in the program, which should trigger notifications
func Error() MessageType {
	return MessageType{Name: "ERROR"}
}

// Warning for logging minor faults or bugs in the program
func Warning() MessageType {
	return MessageType{Name: "WARNING"}
}

// OK for general purpose logging
func OK() MessageType {
	return MessageType{Name: "OK"}
}

// ProgramStatus will log and report various warnings/errors
type ProgramStatus interface {
	IsOK() bool
	Log(MessageType, string)
}

// StatusMap is a mapping of program names to their status data
var StatusMap = make(map[string]ProgramStatus, 0)

// StatusStatus will control logging and reporting of status / warnings / errors
// Holy meta batman
var StatusStatus = NewStatus("Status")

// Remote address to forward pings to
var remote string

// NewStatus will create and return a new program status
func NewStatus(name string) ProgramStatus {

	// Check if program status already exists
	s, ok := StatusMap[name]
	if ok {
		log.Println("[WARNING] Status: " + name + " already exists in the StatusMap")
		return s
	}

	s = Status{Name: name}
	StatusMap[name] = s
	return s
}

// IsOK simply returns if the program is OK
func (s Status) IsOK() bool {
	return s.Status != "ERROR"
}

// Log a message of type OK, ERROR, or WARNING
func (s Status) Log(messageType MessageType, message string) {

	// Format and log message
	formattedMessage := fmt.Sprintf("[%s] %s: %s", messageType.Name, s.Name, message)

	// Log based on status type
	switch messageType.Name {
	case "ERROR":
		s.ErrorLog = append(s.ErrorLog, formattedMessage)
		s.Status = "ERROR"
	case "WARNING":
		s.WarningLog = append(s.WarningLog, formattedMessage)
		if s.Status != "ERROR" {
			s.Status = "WARNING"
		}
	case "OK":

	}

	// Set last updated time to now
	s.LastUpdate = time.Now().Format("2006-01-02 15:04:05.999")

	// Always print
	log.Println(formattedMessage)
}

// GetStatus returns all known program statuses
func GetStatus(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(StatusMap)
}

// GetStatusValue returns a specific status value
func GetStatusValue(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	json.NewEncoder(w).Encode(StatusMap[params["name"]])
}

// SetStatus updates or posts a new status of the named program
func SetStatus(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	var newdata MessageType
	_ = json.NewDecoder(r.Body).Decode(&newdata)

	// Ensure values exist
	if newdata.Name == "" {
		StatusStatus.Log(Warning(), "Ignoring erroneous POST missing type field")
		json.NewEncoder(w).Encode("type is a required field.")
		return
	} else if newdata.Message == "" {
		StatusStatus.Log(Warning(), "Ignoring erroneous POST missing message field")
		json.NewEncoder(w).Encode("message is a required field.")
		return
	}

	// Add / update value in global session
	s, ok := StatusMap[params["name"]]
	if !ok {
		// Status does not exist, we should create one
		s = NewStatus(params["name"])
		StatusStatus.Log(OK(), "Created new Status called "+params["name"])
	}

	switch newdata.Name {
	case "ERROR":
		s.Log(Error(), newdata.Message)
	case "WARNING":
		s.Log(Warning(), newdata.Message)
	default: // for OK messages and mispellings / omissions
		s.Log(OK(), newdata.Message)
	}

	StatusStatus.Log(OK(), "Logged external "+newdata.Name+" Status for "+params["name"]+" with message "+newdata.Message)

	// Respond with inserted values
	json.NewEncoder(w).Encode(s)
}

//
// TODO:
// COPIED FROM PINGS.GO, WILL FIX LATER
//

// Ping will fwd to remote server if connected to internet, otherwise will log locally
func Ping(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	// Ensure we have a server (and a DB) to connect to
	if remote != "" {
		onlineResp, err := http.Get("1.1.1.1")

		if err != nil {
			//
			// Log locally
			//
			defer onlineResp.Body.Close()
			StatusStatus.Log(OK(), "Logging "+params["device"]+" to database")

			// Insert into database
			//err := DB.Write(fmt.Sprintf("ping,device=%s ip=\"%s\"", params["device"], params["ip"]))

			if err != nil {
				StatusStatus.Log(Error(), "Error when logging "+params["device"]+" to database: "+err.Error())
			} else {
				StatusStatus.Log(OK(), "Logged "+params["device"]+" to database")
			}

		} else {
			//
			// FWD request to server since we have internet
			//
			StatusStatus.Log(OK(), "Forwarding "+params["device"]+" to server")
			pingResp, err := http.Get(remote + "?name=" + params["device"] + "&local_ip=" + params["ip"])
			if err != nil {
				defer pingResp.Body.Close()
				StatusStatus.Log(Error(), "Error when forwarding ping: "+err.Error())
			}
		}

		json.NewEncoder(w).Encode("OK")
	}

	// Devices will not act on response anyway, anything but 200 is a waste
	json.NewEncoder(w).Encode("OK")
}

// DumpPings is ideally run when reconnected to internet.
// Will dump local pings to remote server
func DumpPings(w http.ResponseWriter, r *http.Request) {

}
