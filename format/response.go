package format

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// JSONResponse for common return value to API
type JSONResponse struct {
	Output interface{} `json:"output,omitempty"`
	Status string      `json:"status,omitempty"`
	OK     bool        `json:"ok"`
	Method string      `json:"method,omitempty"`
	ID     int         `json:"id,omitempty"`
}

// stat for requests, provided they go through our WriteResponse
type stat struct {
	Failures      int           `json:"failures,omitempty"`
	Successes     int           `json:"successes,omitempty"`
	Total         int           `json:"total,omitempty"`
	TotalSize     int           `json:"totalSize,omitempty"`
	SessionValues int           `json:"sessionValues,omitempty"`
	TimeStarted   time.Time     `json:"timeStarted,omitempty"`
	TimeRunning   time.Duration `json:"timeRunning,omitempty"`
}

// Statistics counts various program data
var Statistics stat

func init() {
	Statistics = stat{TimeStarted: time.Now()}
}

// WriteResponse to an http writer, adding extra info and HTTP status as needed
func WriteResponse(w *http.ResponseWriter, response JSONResponse) {
	// Deref writer
	writer := *w

	// Add string Status if it doesn't exist, add appropriate headers
	if response.OK {
		if response.Status == "" {
			response.Status = "success"
		}
		writer.WriteHeader(http.StatusOK)
		Statistics.Successes++
	} else {
		if response.Status == "" {
			response.Status = "fail"
			writer.WriteHeader(http.StatusBadRequest)
		} else if response.Status == "error" {
			writer.WriteHeader(http.StatusNoContent)
		} else {
			writer.WriteHeader(http.StatusBadRequest)
		}
		Statistics.Failures++
	}

	// Update Statistics
	strResponse, _ := json.Marshal(response)
	Statistics.Total++
	Statistics.TotalSize += len(strResponse)

	// Log this to debug
	log.Debug().
		Str("Output", fmt.Sprintf("%v", response.Output)).
		Str("Status", response.Status).
		Bool("OK", response.OK).
		Msg("Full Response:")

	// Write out this response
	json.NewEncoder(writer).Encode(response)
}

// HandleGetStats exports all known stat requests
func HandleGetStats(w http.ResponseWriter, r *http.Request) {
	// Calculate time running
	Statistics.TimeRunning = time.Since(Statistics.TimeStarted)

	// Echo back message
	WriteResponse(&w, JSONResponse{Output: Statistics, OK: true})
}
