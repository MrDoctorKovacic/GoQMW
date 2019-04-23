package streams

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MrDoctorKovacic/MDroid-Core/external/status"
	"github.com/gorilla/mux"
)

// streamData holds the necc values for each stream
type streamData struct {
	Name string `json:"name,omitempty"` // should correspond with the name in the Stream map
	Host string `json:"host,omitempty"`
	Port string `json:"port,omitempty"`
}

// Streams is all the registered video streams
var Streams map[string]streamData

// StreamStatus will control logging and reporting of status / warnings / errors
var StreamStatus = status.NewStatus("Streams")

// GetAllStreams returns all the registered incoming streams
func GetAllStreams(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(Streams)
}

// GetStream returns a specific session value
func GetStream(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	json.NewEncoder(w).Encode(Streams[params["name"]])
}

// RegisterStream creates a new entry in running SQL DB
func RegisterStream(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	decoder := json.NewDecoder(r.Body)
	var newStream streamData
	err := decoder.Decode(&newStream)

	if err != nil {
		StreamStatus.Log(status.Error(), "Error decoding incoming stream data: "+err.Error())
	} else {
		// Add name to the struct
		newStream.Name = params["name"]

		// Append new stream into mapping
		Streams[newStream.Name] = newStream

		// Log success
		StreamStatus.Log(status.OK(), fmt.Sprintf("Registered new stream: %s at %s:%s", newStream.Name, newStream.Host, newStream.Port))
		json.NewEncoder(w).Encode("OK")
	}
}

// RestartStream will attempt to communicate with the host to restart the given stream
// TODO
func RestartStream(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("OK")
}
