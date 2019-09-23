package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/gorilla/mux"
)

// ALPRData holds the plate and percent for each new ALPR value
type ALPRData struct {
	Plate   string  `json:"plate,omitempty"`
	Percent float32 `json:"percent,omitempty"`
}

// ALPRStatus will control logging and reporting of status / warnings / errors
var ALPRStatus = logging.NewStatus("ALPR")

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
		ALPRStatus.Log(logging.OK(), "Responding to POST request for ALPR")
	}

	if err != nil {
		ALPRStatus.Log(logging.Error(), fmt.Sprintf("Error decoding incoming ALPR data: %s", err.Error()))
		return
	}

	// Decode plate/time/etc values
	plate := strings.Replace(params["plate"], " ", "_", -1)
	percent := newplate.Percent

	if plate != "" {
		if Config.DatabaseEnabled {
			// Insert into database
			online, err := Config.DB.Write(fmt.Sprintf("alpr,plate=%s percent=%f", plate, percent))

			if err != nil {
				errorText := fmt.Sprintf("Error writing %s to influx DB: %s", plate, err.Error())
				if online {
					ALPRStatus.Log(logging.Error(), errorText)
				}
				json.NewEncoder(w).Encode(errorText)
				return
			}

			ALPRStatus.Log(logging.OK(), fmt.Sprintf("Logged %s to database", plate))
		}
	} else {
		errorText := fmt.Sprintf("Missing arguments, ignoring post of %s with percent of %f", plate, percent)
		ALPRStatus.Log(logging.Error(), errorText)
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
