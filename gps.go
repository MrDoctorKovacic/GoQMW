package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/MrDoctorKovacic/MDroid-Core/utils"
)

// GPSData holds various data points we expect to receive
type GPSData struct {
	Latitude  string   `json:"latitude,omitempty"`
	Longitude string   `json:"longitude,omitempty"`
	Time      string   `json:"time,omitempty"` // This will help measure latency :)
	Altitude  *float32 `json:"altitude,omitempty"`
	EPV       *float32 `json:"epv,omitempty"`
	EPT       *float32 `json:"ept,omitempty"`
	Speed     *float32 `json:"speed,omitempty"`
	Climb     *float32 `json:"climb,omitempty"`
}

// GPS is the last posted GPS fix
var GPS GPSData

//
// GPS Functions
//

// GetGPSValue returns the latest GPS fix
func GetGPSValue(w http.ResponseWriter, r *http.Request) {
	// Log if requested
	if VerboseOutput {
		SessionStatus.Log(utils.OK(), "Responding to GET request for all GPS values")
	}
	json.NewEncoder(w).Encode(GPS)
}

// SetGPSValue posts a new GPS fix
func SetGPSValue(w http.ResponseWriter, r *http.Request) {
	var newdata GPSData
	_ = json.NewDecoder(r.Body).Decode(&newdata)

	// Log if requested
	if VerboseOutput {
		SessionStatus.Log(utils.OK(), "Responding to POST request for gps values")
	}

	// Prepare new value
	var postingString strings.Builder

	// Update value for global session if the data is newer (not nil)
	// Can't find a better way to go about this
	if newdata.Latitude != "" {
		GPS.Latitude = newdata.Latitude
		postingString.WriteString(fmt.Sprintf("latitude=\"%s\",", newdata.Latitude))
	}
	if newdata.Longitude != "" {
		GPS.Longitude = newdata.Longitude
		postingString.WriteString(fmt.Sprintf("longitude=\"%s\",", newdata.Longitude))
	}
	if newdata.Altitude != nil {
		GPS.Altitude = newdata.Altitude
		log.Println(fmt.Sprintf("%f", *newdata.Altitude))
		postingString.WriteString(fmt.Sprintf("altitude=%f,", *newdata.Altitude))
	}
	if newdata.Speed != nil {
		GPS.Speed = newdata.Speed
		postingString.WriteString(fmt.Sprintf("speed=%f,", *newdata.Speed))
	}
	if newdata.Climb != nil {
		GPS.Climb = newdata.Climb
		postingString.WriteString(fmt.Sprintf("climb=%f,", *newdata.Climb))
	}
	if newdata.Time != "" {
		GPS.Time = newdata.Time
	}
	if newdata.EPV != nil {
		GPS.EPV = newdata.EPV
		postingString.WriteString(fmt.Sprintf("EPV=%f,", *newdata.EPV))
	}
	if newdata.EPT != nil {
		GPS.EPT = newdata.EPT
		postingString.WriteString(fmt.Sprintf("EPT=%f,", *newdata.EPT))
	}

	// Insert into database
	if Config.DatabaseEnabled {
		err := DB.Write(fmt.Sprintf("gps %s", strings.TrimSuffix(postingString.String(), ",")))

		if err != nil {
			SessionStatus.Log(utils.Error(), "Error writing string "+postingString.String()+" to influx DB: "+err.Error())
		} else {
			SessionStatus.Log(utils.OK(), "Logged "+postingString.String()+" to database")
		}
	}

	json.NewEncoder(w).Encode("OK")
}
