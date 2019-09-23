package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/logging"
)

// GPSData holds various data points we expect to receive
type GPSData struct {
	Latitude  string `json:"latitude,omitempty"`
	Longitude string `json:"longitude,omitempty"`
	Time      string `json:"time,omitempty"` // This will help measure latency :)
	Altitude  string `json:"altitude,omitempty"`
	EPV       string `json:"epv,omitempty"`
	EPT       string `json:"ept,omitempty"`
	Speed     string `json:"speed,omitempty"`
	Climb     string `json:"climb,omitempty"`
	Course    string `json:"course,omitempty"`
}

// GPS is the last posted GPS fix
var GPS GPSData

//
// GPS Functions
//

// GetGPSValue returns the latest GPS fix
func GetGPSValue(w http.ResponseWriter, r *http.Request) {
	// Log if requested
	if Config.VerboseOutput {
		SessionStatus.Log(logging.OK(), "Responding to GET request for all GPS values")
	}
	json.NewEncoder(w).Encode(GPS)
}

// SetGPSValue posts a new GPS fix
func SetGPSValue(w http.ResponseWriter, r *http.Request) {
	var newdata GPSData
	_ = json.NewDecoder(r.Body).Decode(&newdata)

	// Log if requested
	if Config.VerboseOutput {
		SessionStatus.Log(logging.OK(), "Responding to POST request for gps values")
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
	if newdata.Altitude != "" {
		GPS.Altitude = newdata.Altitude
		convFloat, _ := strconv.ParseFloat(newdata.Altitude, 32)
		postingString.WriteString(fmt.Sprintf("altitude=%f,", convFloat))
	}
	if newdata.Speed != "" {
		GPS.Speed = newdata.Speed
		convFloat, _ := strconv.ParseFloat(newdata.Speed, 32)
		postingString.WriteString(fmt.Sprintf("speed=%f,", convFloat))
	}
	if newdata.Climb != "" {
		GPS.Climb = newdata.Climb
		convFloat, _ := strconv.ParseFloat(newdata.Climb, 32)
		postingString.WriteString(fmt.Sprintf("climb=%f,", convFloat))
	}
	if newdata.Time == "" {
		newdata.Time = time.Now().In(Timezone).Format("2006-01-02 15:04:05.999")
	}
	GPS.Time = newdata.Time
	if newdata.EPV != "" {
		GPS.EPV = newdata.EPV
		postingString.WriteString(fmt.Sprintf("EPV=%s,", newdata.EPV))
	}
	if newdata.EPT != "" {
		GPS.EPT = newdata.EPT
		postingString.WriteString(fmt.Sprintf("EPT=%s,", newdata.EPT))
	}
	if newdata.Course != "" {
		GPS.Course = newdata.Course
		convFloat, _ := strconv.ParseFloat(newdata.Course, 32)
		postingString.WriteString(fmt.Sprintf("Course=%f,", convFloat))
	}

	// Insert into database
	if Config.DatabaseEnabled {
		online, err := Config.DB.Write(fmt.Sprintf("gps %s", strings.TrimSuffix(postingString.String(), ",")))

		if err != nil && online {
			SessionStatus.Log(logging.Error(), fmt.Sprintf("Error writing string %s to influx DB: %s", postingString.String(), err.Error()))
		} else if Config.VerboseOutput {
			SessionStatus.Log(logging.OK(), fmt.Sprintf("Logged %s to database", postingString.String()))
		}
	}

	json.NewEncoder(w).Encode("OK")
}
