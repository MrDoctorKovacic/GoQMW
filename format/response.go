package format

import (
	"encoding/json"
	"fmt"
	"net/http"

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
	failures  int
	successes int
	total     int
	totalSize int
}

var statistics stat

func init() {
	statistics = stat{}
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
		statistics.successes++
	} else {
		if response.Status == "" {
			response.Status = "fail"
			writer.WriteHeader(http.StatusBadRequest)
		} else if response.Status == "error" {
			writer.WriteHeader(http.StatusNoContent)
		} else {
			writer.WriteHeader(http.StatusBadRequest)
		}
		statistics.failures++
	}

	// Update statistics
	strResponse, _ := json.Marshal(response)
	statistics.total++
	statistics.totalSize += len(strResponse)

	// Log this to debug
	log.Debug().
		Str("Output", fmt.Sprintf("%v", response.Output)).
		Str("Status", response.Status).
		Bool("OK", response.OK).
		Msg("Full Response:")

	// Write out this response
	json.NewEncoder(writer).Encode(response)
}
