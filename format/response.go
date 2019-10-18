package format

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/influx"
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
	TotalSize     int64         `json:"totalSize,omitempty"`
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
func WriteResponse(w *http.ResponseWriter, r *http.Request, response JSONResponse) {
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
	// Add the request and response sizes together
	Statistics.TotalSize += int64(len(strResponse)) + r.ContentLength

	intOK := 0
	if response.OK {
		intOK = 1
	}

	// Log this to our DB
	if influx.DB != nil {
		online, err := influx.DB.Write(fmt.Sprintf("requests,method=\"%s\",path=\"%s\" ok=%d", r.Method, r.URL.Path, intOK))
		if err != nil {
			errorText := fmt.Sprintf("Error writing method=%s, path=%s to influx DB: %s", r.Method, r.URL.Path, err.Error())
			// Only spam our log if Influx is online
			if online {
				log.Error().Msg(errorText)
			}
		}
		log.Debug().Msg(fmt.Sprintf("Logged request to %s in DB", r.URL.Path))
	}

	// Log this to debug
	log.Debug().
		Str("Path", r.URL.Path).
		Str("Method", r.Method).
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
	WriteResponse(&w, r, JSONResponse{Output: Statistics, OK: true})
}
