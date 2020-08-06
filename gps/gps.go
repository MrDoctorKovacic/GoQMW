// Package gps implements session values regarding GPS and timezone Mods
package gps

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bradfitz/latlong"
	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/db"
	"github.com/qcasey/MDroid-Core/format/response"
	"github.com/qcasey/MDroid-Core/mqtt"
	"github.com/qcasey/MDroid-Core/settings"
	"github.com/rs/zerolog/log"
)

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

var (
	timezone   *time.Location
	currentFix Fix
	lastFix    Fix
	mutex      sync.Mutex
)

func init() {
	var err error
	timezone, err = time.LoadLocation("America/Los_Angeles")
	if err != nil {
		log.Error().Msg("Could not load default timezone")
		return
	}
}

// Setup timezone as per module standards
func Setup(router *mux.Router) {
	if settings.Data.IsSet("mdroid.timezone") {
		loc, err := time.LoadLocation(settings.Data.GetString("mdroid.timezone"))
		if err != nil {
			timezone, _ = time.LoadLocation("UTC")
			return
		}

		timezone = loc
		return
	}

	// Timezone is not set in config
	timezone, _ = time.LoadLocation("UTC")
	log.Info().Msgf("Set timezone to %s", timezone.String())

	//
	// GPS Routes
	//
	router.HandleFunc("/session/gps", HandleGet).Methods("GET")
	router.HandleFunc("/session/gps", HandleSet).Methods("POST")
	router.HandleFunc("/session/timezone", func(w http.ResponseWriter, r *http.Request) {
		response := response.JSONResponse{Output: GetTimezone(), OK: true}
		response.Write(&w, r)
	}).Methods("GET")
}

//
// GPS Functions
//

// HandleGet returns the latest GPS fix
func HandleGet(w http.ResponseWriter, r *http.Request) {
	data := Get()
	response.WriteNew(&w, r, response.JSONResponse{Output: data, OK: true})
}

// Get returns the latest GPS fix
func Get() Fix {
	// Log if requested
	mutex.Lock()
	gpsFix := currentFix
	mutex.Unlock()

	return gpsFix
}

// GetTimezone returns the latest GPS timezone recorded
func GetTimezone() *time.Location {
	// Log if requested
	mutex.Lock()
	timezone := timezone
	mutex.Unlock()

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
	response.WriteNew(&w, r, response.JSONResponse{Output: "OK", OK: true})
}

func fixIsSignifigantlyDifferent(oldFix string, newFix string) bool {
	if oldFloat, err := strconv.ParseFloat(oldFix, 64); err == nil {
		if newFloat, err := strconv.ParseFloat(newFix, 64); err == nil {
			if math.Abs(oldFloat-newFloat) > 0.000001 {
				return true
			}
		}
	}
	return false
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

	mutex.Lock()
	// Update Location fixes
	lastFix = currentFix
	currentFix = newdata
	mutex.Unlock()

	// Update timezone information with new GPS fix
	processTimezone()

	// Initial posting string for Influx DB
	postingString.WriteString(fmt.Sprintf("latitude=\"%s\",", newdata.Latitude))
	postingString.WriteString(fmt.Sprintf("longitude=\"%s\",", newdata.Longitude))

	if fixIsSignifigantlyDifferent(lastFix.Latitude, newdata.Latitude) {
		go mqtt.Publish("gps/latitude", newdata.Latitude, true)
	}
	if fixIsSignifigantlyDifferent(lastFix.Longitude, newdata.Longitude) {
		go mqtt.Publish("gps/longitude", newdata.Longitude, true)
	}

	// Append posting strings based on what GPS information was posted
	if convFloat, err := strconv.ParseFloat(newdata.Altitude, 32); err == nil {
		if lastFix.Altitude != newdata.Altitude {
			go mqtt.Publish("gps/altitude", convFloat, true)
		}
		postingString.WriteString(fmt.Sprintf("altitude=%f,", convFloat))
	}
	if convFloat, err := strconv.ParseFloat(newdata.Speed, 32); err == nil {
		if lastFix.Speed != newdata.Speed {
			go mqtt.Publish("gps/speed", convFloat, true)
		}
		postingString.WriteString(fmt.Sprintf("speed=%f,", convFloat))
	}
	if convFloat, err := strconv.ParseFloat(newdata.Climb, 32); err == nil {
		if lastFix.Climb != newdata.Climb {
			go mqtt.Publish("gps/climb", convFloat, true)
		}
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
	mutex.Lock()
	latFloat, err1 := strconv.ParseFloat(currentFix.Latitude, 64)
	longFloat, err2 := strconv.ParseFloat(currentFix.Longitude, 64)
	mutex.Unlock()

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

	mutex.Lock()
	timezone = newTimezone
	mutex.Unlock()
}
