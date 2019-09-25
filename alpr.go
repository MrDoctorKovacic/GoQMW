package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/gorilla/mux"
)

// alprData holds the plate and percent for each new ALPR value
type alprData struct {
	Plate   string  `json:"plate,omitempty"`
	Percent float32 `json:"percent,omitempty"`
}

// alprStatus will control logging and reporting of status / warnings / errors
var alprStatus = logging.NewStatus("ALPR")

//
// ALPR Functions
//

// logALPR creates a new entry in running SQL DB
func logALPR(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	decoder := json.NewDecoder(r.Body)
	var newplate alprData
	err := decoder.Decode(&newplate)

	// Log if requested
	if Config.VerboseOutput {
		alprStatus.Log(logging.OK(), "Responding to POST request for ALPR")
	}

	if err != nil {
		alprStatus.Log(logging.Error(), fmt.Sprintf("Error decoding incoming ALPR data: %s", err.Error()))
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
					alprStatus.Log(logging.Error(), errorText)
				}
				json.NewEncoder(w).Encode(formatting.JSONResponse{Output: errorText, Status: "fail", OK: false})
				return
			}

			alprStatus.Log(logging.OK(), fmt.Sprintf("Logged %s to database", plate))
		}
	} else {
		errorText := fmt.Sprintf("Missing arguments, ignoring post of %s with percent of %f", plate, percent)
		alprStatus.Log(logging.Error(), errorText)
		json.NewEncoder(w).Encode(formatting.JSONResponse{Output: errorText, Status: "fail", OK: false})
		return
	}

	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "OK", Status: "success", OK: true})
}

// restartALPR posts remote device to restart ALPR service
// TODO
func restartALPR(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "OK", Status: "success", OK: true})
}
