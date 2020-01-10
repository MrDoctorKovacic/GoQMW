// Package gps implements session values regarding GPS and timezone locations
package gps

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/qcasey/MDroid-Core/db"
	"github.com/qcasey/MDroid-Core/format"
	"github.com/bradfitz/latlong"
	"github.com/rs/zerolog/log"
)

// Loc contains GPS meta data and other location information
type Loc struct {
	Timezone   *time.Location
	CurrentFix Fix
	LastFix    Fix
	Mutex      sync.Mutex
}

// Fix holds various data points we expect to receive
type Fix struct {
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

// status will control logging and reporting of status / warnings / errors
var (
	Location *Loc
)

func init() {
	timezone, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		log.Error().Msg("Could not load default timezone")
		return
	}
	Location = &Loc{Timezone: timezone} // use logging default timezone
}

//
// GPS Functions
//

// HandleGet returns the latest GPS fix
func HandleGet(w http.ResponseWriter, r *http.Request) {
	data := Get()
	format.WriteResponse(&w, r, format.JSONResponse{Output: data, OK: true})
}

// Get returns the latest GPS fix
func Get() Fix {
	// Log if requested
	Location.Mutex.Lock()
	gpsFix := Location.CurrentFix
	Location.Mutex.Unlock()

	return gpsFix
}

// GetTimezone returns the latest GPS timezone recorded
func GetTimezone() *time.Location {
	// Log if requested
	Location.Mutex.Lock()
	timezone := Location.Timezone
	Location.Mutex.Unlock()

	return timezone
}

// HandleSet posts a new GPS fix
func HandleSet(w http.ResponseWriter, r *http.Request) {
	var newdata Fix
	if err := json.NewDecoder(r.Body).Decode(&newdata); err != nil {
		log.Error().Msg(err.Error())
		return
	}
	postingString := Set(newdata)

	// Insert into database
	if postingString != "" && db.DB != nil {
		err := db.DB.Write(fmt.Sprintf("gps %s", strings.TrimSuffix(postingString, ",")))
		if err != nil && db.DB.Started {
			log.Error().Msgf("Error writing string %s to influx DB: %s", postingString, err.Error())
			return
		}
		log.Debug().Msgf("Logged %s to database", postingString)
	}
	format.WriteResponse(&w, r, format.JSONResponse{Output: "OK", OK: true})
}

// Set posts a new GPS fix
func Set(newdata Fix) string {
	// Update value for global session if the data is newer
	if newdata.Latitude == "" && newdata.Longitude == "" {
		log.Debug().Msg("Not inserting new GPS fix, no new Lat or Long")
		return ""
	}

	// Prepare new value
	var postingString strings.Builder

	Location.Mutex.Lock()
	// Update Loc fixes
	Location.LastFix = Location.CurrentFix
	Location.CurrentFix = newdata
	Location.Mutex.Unlock()

	// Update timezone information with new GPS fix
	processTimezone()

	// Initial posting string for Influx DB
	postingString.WriteString(fmt.Sprintf("latitude=\"%s\",", newdata.Latitude))
	postingString.WriteString(fmt.Sprintf("longitude=\"%s\",", newdata.Longitude))

	// Append posting strings based on what GPS information was posted
	if convFloat, err := strconv.ParseFloat(newdata.Altitude, 32); err == nil {
		postingString.WriteString(fmt.Sprintf("altitude=%f,", convFloat))
	}
	if convFloat, err := strconv.ParseFloat(newdata.Speed, 32); err == nil {
		postingString.WriteString(fmt.Sprintf("speed=%f,", convFloat))
	}
	if convFloat, err := strconv.ParseFloat(newdata.Climb, 32); err == nil {
		postingString.WriteString(fmt.Sprintf("climb=%f,", convFloat))
	}
	if newdata.Time == "" {
		newdata.Time = time.Now().In(GetTimezone()).Format("2006-01-02 15:04:05.999")
	}
	if newdata.EPV != "" {
		postingString.WriteString(fmt.Sprintf("EPV=%s,", newdata.EPV))
	}
	if newdata.EPT != "" {
		postingString.WriteString(fmt.Sprintf("EPT=%s,", newdata.EPT))
	}
	if convFloat, err := strconv.ParseFloat(newdata.Course, 32); err == nil {
		postingString.WriteString(fmt.Sprintf("Course=%f,", convFloat))
	}

	return postingString.String()
}

// Parses GPS coordinates into a time.Location timezone
// On OpenWRT, this requires the zoneinfo-core and zoneinfo-northamerica (or other relevant locations) packages
func processTimezone() {
	Location.Mutex.Lock()
	latFloat, err1 := strconv.ParseFloat(Location.CurrentFix.Latitude, 64)
	longFloat, err2 := strconv.ParseFloat(Location.CurrentFix.Longitude, 64)
	Location.Mutex.Unlock()

	if err1 != nil {
		log.Error().Msgf("Error converting lat into float64: %s", err1.Error())
		return
	}
	if err2 != nil {
		log.Error().Msgf("Error converting long into float64: %s", err2.Error())
		return
	}

	timezoneName := latlong.LookupZoneName(latFloat, longFloat)
	newTimezone, err := time.LoadLocation(timezoneName)
	if err != nil {
		log.Error().Msgf("Error parsing lat long into location: %s", err.Error())
		return
	}

	Location.Mutex.Lock()
	Location.Timezone = newTimezone
	Location.Mutex.Unlock()
}

// SetupTimezone loads settings data into timezone
func SetupTimezone(configAddr *map[string]string) {
	configMap := *configAddr

	if timezoneLocation, usingTimezone := configMap["TIMEZONE"]; usingTimezone {
		loc, err := time.LoadLocation(timezoneLocation)
		if err != nil {
			Location.Timezone, _ = time.LoadLocation("UTC")
			return
		}

		Location.Timezone = loc
		return
	}

	// Timezone is not set in config
	Location.Timezone, _ = time.LoadLocation("UTC")
	log.Info().Msgf("Set timezone to %s", Location.Timezone.String())
}
