package logging

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Status of a particular program
// Keep some logs in memory
type Status struct {
	Name       string // should match the name in the statusMap
	Status     string
	LastUpdate string
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

// statusMap is a mapping of program names to their status data
var (
	statusMap         map[string]ProgramStatus
	statusLock        sync.Mutex
	status            ProgramStatus
	RemotePingAddress string
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	statusMap = make(map[string]ProgramStatus, 0)
	status = NewStatus("Status")
}

// NewStatus will create and return a new program status
func NewStatus(name string) ProgramStatus {

	// Check if program status already exists
	statusLock.Lock()
	s, ok := statusMap[name]
	statusLock.Unlock()
	if ok {
		log.Warn().Msg("Status: " + name + " already exists in the statusMap")
		return s
	}

	s = &Status{Name: name}
	statusLock.Lock()
	statusMap[name] = s
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
	formattedMessage := fmt.Sprintf("%s: %s", s.Name, message)

	// Log based on status type
	switch messageType.Name {
	case "ERROR":
		log.Error().Msg(formattedMessage)
		s.Status = "ERROR"
	case "WARNING":
		log.Warn().Msg(formattedMessage)
		if s.Status != "ERROR" {
			s.Status = "WARNING"
		}
	case "OK":
		log.Info().Msg(formattedMessage)

	}
}

// Get returns all known program statuses
func Get(w http.ResponseWriter, r *http.Request) {
	statusLock.Lock()
	json.NewEncoder(w).Encode(statusMap)
	statusLock.Unlock()
}

// GetValue returns a specific status value
func GetValue(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	statusLock.Lock()
	json.NewEncoder(w).Encode(statusMap[params["name"]])
	statusLock.Unlock()
}

// Set updates or posts a new status of the named program
func Set(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	var newdata MessageType
	err := json.NewDecoder(r.Body).Decode(&newdata)
	if err != nil {
		status.Log(Error(), err.Error())
		return
	}

	// Ensure values exist
	if newdata.Name == "" {
		status.Log(Warning(), "Ignoring erroneous POST missing type field")
		json.NewEncoder(w).Encode("type is a required field.")
		return
	} else if newdata.Message == "" {
		status.Log(Warning(), "Ignoring erroneous POST missing message field")
		json.NewEncoder(w).Encode("message is a required field.")
		return
	}

	// Add / update value in global session
	statusLock.Lock()
	s, ok := statusMap[params["name"]]
	statusLock.Unlock()
	if !ok {
		// Status does not exist, we should create one
		s = NewStatus(params["name"])
		status.Log(OK(), fmt.Sprintf("Created new Status called %s", params["name"]))
	}

	switch newdata.Name {
	case "ERROR":
		s.Log(Error(), newdata.Message)
	case "WARNING":
		s.Log(Warning(), newdata.Message)
	default: // for OK messages and mispellings / omissions
		s.Log(OK(), newdata.Message)
	}

	status.Log(OK(), fmt.Sprintf("Logged external %s Status for %s with message %s", newdata.Name, params["name"], newdata.Message))

	// Respond with inserted values
	json.NewEncoder(w).Encode(s)
}
