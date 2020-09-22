package session

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/internal/core"
	"github.com/rs/zerolog/log"
)

// Package holds the Package and last update info for each session value
type Package struct {
	Name       string      `json:"name,omitempty"`
	Value      interface{} `json:"value,omitempty"`
	LastUpdate string      `json:"lastUpdate,omitempty"`
	date       time.Time
	Quiet      bool `json:"quiet,omitempty"`
}

// Set updates or posts a new session value to the common session
func Set(c *core.Core) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)

		// Default to NOT OK response
		response := core.JSONResponse{OK: false}

		if err != nil {
			log.Error().Msgf("Error reading body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}

		// Put body back
		r.Body.Close() //  must close
		r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

		if len(body) == 0 {
			response.Output = "Error: Empty body"
			response.Write(&w, r)
			return
		}

		params := mux.Vars(r)
		var newdata Package

		if err = json.NewDecoder(r.Body).Decode(&newdata); err != nil {
			log.Error().Msgf("Error decoding incoming JSON:\n%s", err.Error())
			response.Output = err.Error()
			response.Write(&w, r)
			return
		}

		// Call the setter
		newdata.Name = params["name"]
		c.Session.Set(params["name"], newdata.Value)

		// Craft OK response
		response.OK = true
		response.Output = newdata

		response.Write(&w, r)
	}
}
