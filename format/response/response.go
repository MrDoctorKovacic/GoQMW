package response

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

// stat for requests, provided they go through our Write
type stat struct {
	Failures      int       `json:"failures,omitempty"`
	Successes     int       `json:"successes,omitempty"`
	Total         int       `json:"total,omitempty"`
	TotalSize     int64     `json:"totalSize,omitempty"`
	SessionValues int       `json:"sessionValues,omitempty"`
	TimeStarted   time.Time `json:"timeStarted,omitempty"`
	TimeRunning   float64   `json:"timeRunning,omitempty"`
}

// Statistics counts various program data
var Statistics stat

func init() {
	Statistics = stat{TimeStarted: time.Now()}
}

// Write to an http writer, adding extra info and HTTP status as needed
func (response *JSONResponse) Write(w *http.ResponseWriter, r *http.Request) {
	// Deref writer
	writer := *w

	writer.Header().Set("Content-Type", "application/json")

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
	requestSize := int64(len(strResponse)) + r.ContentLength
	Statistics.TotalSize += requestSize

	/*intOK := 0
	if response.OK {
		intOK = 1
	}*/

	// Log this to our DB
	/*
		if db.DB != nil {
			err := db.DB.Insert("requests", map[string]interface{}{"method": r.Method, "path": r.URL.Path}, map[string]interface{}{"ok": intOK, "size": requestSize})
			if err != nil {
				errorText := fmt.Sprintf("Error writing method=%s, path=%s to influx DB: %s", r.Method, r.URL.Path, err.Error())
				// Only spam our log if Influx is online
				if db.DB.Started {
					log.Error().Msg(errorText)
				}
			}
			log.Debug().Msgf("Logged request to %s in DB", r.URL.Path)
		}*/

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

// WriteNew exports all known stat requests
func WriteNew(w *http.ResponseWriter, r *http.Request, response JSONResponse) {
	// Echo back message
	response.Write(w, r)
}

// HandleGetStats exports all known stat requests
func HandleGetStats(w http.ResponseWriter, r *http.Request) {
	// Calculate time running
	Statistics.TimeRunning = time.Since(Statistics.TimeStarted).Seconds()

	// Echo back message
	WriteNew(&w, r, JSONResponse{Output: Statistics, OK: true})
}
