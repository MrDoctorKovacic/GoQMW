package gps

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/bradfitz/latlong"
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
	status   logging.ProgramStatus
	Location *Loc
)

func init() {
	status = logging.NewStatus("GPS")
	Location = &Loc{Timezone: logging.Timezone} // use logging default timezone
}

//
// GPS Functions
//

// HandleGet returns the latest GPS fix
func HandleGet(w http.ResponseWriter, r *http.Request) {
	// Log if requested
	//status.Log(logging.Warning(), "Responding to get request.")
	data := Get()
	if data.Latitude == "" && data.Longitude == "" {
		json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "GPS data is empty", Status: "fail", OK: false})
	} else {
		json.NewEncoder(w).Encode(formatting.JSONResponse{Output: data, Status: "success", OK: true})
	}
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
	err := json.NewDecoder(r.Body).Decode(&newdata)
	if err != nil {
		status.Log(logging.Error(), err.Error())
		return
	}
	postingString := Set(newdata)

	// Insert into database
	if postingString != "" && settings.Config.DB != nil {
		online, err := settings.Config.DB.Write(fmt.Sprintf("gps %s", strings.TrimSuffix(postingString, ",")))

		if err != nil && online {
			status.Log(logging.Error(), fmt.Sprintf("Error writing string %s to influx DB: %s", postingString, err.Error()))
		} else {
			status.Log(logging.Debug(), fmt.Sprintf("Logged %s to database", postingString))
		}
	}
}

// Set posts a new GPS fix
func Set(newdata Fix) string {
	// Update value for global session if the data is newer
	if newdata.Latitude == "" && newdata.Longitude == "" {
		status.Log(logging.Warning(), "Not inserting new GPS fix, no new Lat or Long")
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
	if newdata.Altitude != "" {
		convFloat, err := strconv.ParseFloat(newdata.Altitude, 32)
		if err == nil {
			postingString.WriteString(fmt.Sprintf("altitude=%f,", convFloat))
		}
	}
	if newdata.Speed != "" {
		convFloat, err := strconv.ParseFloat(newdata.Speed, 32)
		if err == nil {
			postingString.WriteString(fmt.Sprintf("speed=%f,", convFloat))
		}
	}
	if newdata.Climb != "" {
		convFloat, err := strconv.ParseFloat(newdata.Climb, 32)
		if err == nil {
			postingString.WriteString(fmt.Sprintf("climb=%f,", convFloat))
		}
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
	if newdata.Course != "" {
		convFloat, err := strconv.ParseFloat(newdata.Course, 32)
		if err == nil {
			postingString.WriteString(fmt.Sprintf("Course=%f,", convFloat))
		}
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
		status.Log(logging.Error(), fmt.Sprintf("Error converting lat into float64: %s", err1.Error()))
		return
	}
	if err2 != nil {
		status.Log(logging.Error(), fmt.Sprintf("Error converting long into float64: %s", err2.Error()))
		return
	}

	timezoneName := latlong.LookupZoneName(latFloat, longFloat)
	newTimezone, err := time.LoadLocation(timezoneName)
	if err != nil {
		status.Log(logging.Error(), fmt.Sprintf("Error parsing lat long into location: %s", err.Error()))
		return
	}

	// Set logging timezone
	logging.SetTimezone(newTimezone)

	Location.Mutex.Lock()
	Location.Timezone = newTimezone
	Location.Mutex.Unlock()
}
