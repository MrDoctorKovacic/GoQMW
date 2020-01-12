// Package gps implements session values regarding GPS and timezone Mods
package gps

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bradfitz/latlong"
	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/db"
	"github.com/qcasey/MDroid-Core/format"
	"github.com/rs/zerolog/log"
)

// Location contains GPS meta data and other Mod information
type Location struct {
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

// Mod is the module implementation
var Mod *Location

func init() {
	timezone, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		log.Error().Msg("Could not load default timezone")
		return
	}
	Mod = &Location{Timezone: timezone} // use logging default timezone
}

// Setup timezone as per module standards
func (*Location) Setup(configAddr *map[string]string) {
	configMap := *configAddr

	if timezoneMod, usingTimezone := configMap["TIMEZONE"]; usingTimezone {
		loc, err := time.LoadLocation(timezoneMod)
		if err != nil {
			Mod.Timezone, _ = time.LoadLocation("UTC")
			return
		}

		Mod.Timezone = loc
		return
	}

	// Timezone is not set in config
	Mod.Timezone, _ = time.LoadLocation("UTC")
	log.Info().Msgf("Set timezone to %s", Mod.Timezone.String())
}

// SetRoutes implements router aggregate function
func (*Location) SetRoutes(router *mux.Router) {
	//
	// GPS Routes
	//
	router.HandleFunc("/session/gps", HandleGet).Methods("GET")
	router.HandleFunc("/session/gps", HandleSet).Methods("POST")
	router.HandleFunc("/session/timezone", func(w http.ResponseWriter, r *http.Request) {
		response := format.JSONResponse{Output: GetTimezone(), OK: true}
		format.WriteResponse(&w, r, response)
	}).Methods("GET")
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
	Mod.Mutex.Lock()
	gpsFix := Mod.CurrentFix
	Mod.Mutex.Unlock()

	return gpsFix
}

// GetTimezone returns the latest GPS timezone recorded
func GetTimezone() *time.Location {
	// Log if requested
	Mod.Mutex.Lock()
	timezone := Mod.Timezone
	Mod.Mutex.Unlock()

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

	Mod.Mutex.Lock()
	// Update Location fixes
	Mod.LastFix = Mod.CurrentFix
	Mod.CurrentFix = newdata
	Mod.Mutex.Unlock()

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

// Parses GPS coordinates into a time.Mod timezone
// On OpenWRT, this requires the zoneinfo-core and zoneinfo-northamerica (or other relevant Mods) packages
func processTimezone() {
	Mod.Mutex.Lock()
	latFloat, err1 := strconv.ParseFloat(Mod.CurrentFix.Latitude, 64)
	longFloat, err2 := strconv.ParseFloat(Mod.CurrentFix.Longitude, 64)
	Mod.Mutex.Unlock()

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
		log.Error().Msgf("Error parsing lat long into Mod: %s", err.Error())
		return
	}

	Mod.Mutex.Lock()
	Mod.Timezone = newTimezone
	Mod.Mutex.Unlock()
}
