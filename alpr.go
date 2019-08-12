package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/MrDoctorKovacic/MDroid-Core/status"
	"github.com/gorilla/mux"
)

// ALPRData holds the plate and percent for each new ALPR value
type ALPRData struct {
	Plate   string `json:"plate,omitempty"`
	Percent int    `json:"percent,omitempty"`
}

// ALPRStatus will control logging and reporting of status / warnings / errors
var ALPRStatus = status.NewStatus("ALPR")

//
// ALPR Functions
//

// LogALPR creates a new entry in running SQL DB
func LogALPR(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	decoder := json.NewDecoder(r.Body)
	var newplate ALPRData
	err := decoder.Decode(&newplate)

	// Log if requested
	if Config.VerboseOutput {
		ALPRStatus.Log(status.OK(), "Responding to POST request for ALPR")
	}

	if err != nil {
		ALPRStatus.Log(status.Error(), fmt.Sprintf("Error decoding incoming ALPR data: %s", err.Error()))
		return
	}

	// Decode plate/time/etc values
	plate := strings.Replace(params["plate"], " ", "_", -1)
	percent := newplate.Percent

	if plate != "" {
		if Config.DatabaseEnabled {
			// Insert into database
			err := DB.Write(fmt.Sprintf("alpr,plate=%s percent=%d", plate, percent))

			if err != nil {
				errorText := fmt.Sprintf("Error writing %s to influx DB: %s", plate, err.Error())
				ALPRStatus.Log(status.Error(), errorText)
				json.NewEncoder(w).Encode(errorText)
				return
			}

			ALPRStatus.Log(status.OK(), fmt.Sprintf("Logged %s to database", plate))
		}
	} else {
		errorText := fmt.Sprintf("Missing arguments, ignoring post of %s with percent of %d", plate, percent)
		ALPRStatus.Log(status.Error(), errorText)
		json.NewEncoder(w).Encode(errorText)
		return
	}

	json.NewEncoder(w).Encode("OK")
}

// RestartALPR posts remote device to restart ALPR service
// TODO
func RestartALPR(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("OK")
}
