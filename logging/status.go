package logging

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
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
	Mutex      sync.Mutex
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

// Status Lock
var statusLock sync.Mutex

// StatusStatus will control logging and reporting of status / warnings / errors
// Holy meta batman
var StatusStatus = NewStatus("Status")

// RemotePingAddress to forward pings to
var RemotePingAddress string

// NewStatus will create and return a new program status
func NewStatus(name string) ProgramStatus {

	// Check if program status already exists
	statusLock.Lock()
	s, ok := StatusMap[name]
	statusLock.Unlock()
	if ok {
		log.Println("[WARNING] Status: " + name + " already exists in the StatusMap")
		return s
	}

	s = &Status{Name: name}
	statusLock.Lock()
	StatusMap[name] = s
	statusLock.Unlock()
	return s
}

// IsOK simply returns if the program is OK
func (s *Status) IsOK() bool {
	return s.Status != "ERROR"
}

// Log a message of type OK, ERROR, or WARNING
func (s *Status) Log(messageType MessageType, message string) {

	// Format and log message
	formattedMessage := fmt.Sprintf("[%s] %s: %s", messageType.Name, s.Name, message)

	// Set last updated time to now
	s.LastUpdate = time.Now().In(GetTimezone()).Format("2006-01-02 15:04:05.999")

	// Log based on status type
	s.Mutex.Lock()
	switch messageType.Name {
	case "ERROR":
		s.ErrorLog = append(s.ErrorLog, fmt.Sprintf("%s %s", s.LastUpdate, formattedMessage))
		s.Status = "ERROR"
	case "WARNING":
		s.WarningLog = append(s.WarningLog, fmt.Sprintf("%s %s", s.LastUpdate, formattedMessage))
		if s.Status != "ERROR" {
			s.Status = "WARNING"
		}
	case "OK":

	}
	s.Mutex.Unlock()

	// Always print
	log.Println(formattedMessage)
}

// GetStatus returns all known program statuses
func GetStatus(w http.ResponseWriter, r *http.Request) {
	statusLock.Lock()
	json.NewEncoder(w).Encode(StatusMap)
	statusLock.Unlock()
}

// GetStatusValue returns a specific status value
func GetStatusValue(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	statusLock.Lock()
	json.NewEncoder(w).Encode(StatusMap[params["name"]])
	statusLock.Unlock()
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
	statusLock.Lock()
	s, ok := StatusMap[params["name"]]
	statusLock.Unlock()
	if !ok {
		// Status does not exist, we should create one
		s = NewStatus(params["name"])
		StatusStatus.Log(OK(), fmt.Sprintf("Created new Status called %s", params["name"]))
	}

	switch newdata.Name {
	case "ERROR":
		s.Log(Error(), newdata.Message)
	case "WARNING":
		s.Log(Warning(), newdata.Message)
	default: // for OK messages and mispellings / omissions
		s.Log(OK(), newdata.Message)
	}

	StatusStatus.Log(OK(), fmt.Sprintf("Logged external %s Status for %s with message %s", newdata.Name, params["name"], newdata.Message))

	// Respond with inserted values
	json.NewEncoder(w).Encode(s)
}
