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
)

// Location contains GPS meta data and other location information
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

// gpsStatus will control logging and reporting of status / warnings / errors
var gpsStatus = logging.NewStatus("GPS")

//
// GPS Functions
//

// HandleGet returns the latest GPS fix
func (loc *Location) HandleGet(w http.ResponseWriter, r *http.Request) {
	// Log if requested
	loc.Mutex.Lock()
	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: loc.Get(), Status: "success", OK: true})
	loc.Mutex.Unlock()
}

// Get returns the latest GPS fix
func (loc *Location) Get() Fix {
	// Log if requested
	loc.Mutex.Lock()
	gpsFix := loc.CurrentFix
	loc.Mutex.Unlock()

	return gpsFix
}

// Set posts a new GPS fix
func (loc *Location) Set(newdata Fix) string {
	// Prepare new value
	var postingString strings.Builder

	// Update value for global session if the data is newer (not nil)
	// Can't find a better way to go about this
	loc.Mutex.Lock()
	if newdata.Latitude == "" && newdata.Longitude == "" {
		gpsStatus.Log(logging.Warning(), "Not inserting new GPS fix, no new Lat or Long")
		return
	}

	// Update Loc fixes
	loc.LastFix = loc.CurrentFix
	loc.CurrentFix = newdata

	// Update timezone information with new GPS fix
	loc.processTimezone()

	// Initial posting string for Influx DB
	postingString.WriteString(fmt.Sprintf("latitude=\"%s\",", newdata.Latitude))
	postingString.WriteString(fmt.Sprintf("longitude=\"%s\",", newdata.Longitude))

	// Append posting strings based on what GPS information was posted
	if newdata.Altitude != "" {
		convFloat, _ := strconv.ParseFloat(newdata.Altitude, 32)
		postingString.WriteString(fmt.Sprintf("altitude=%f,", convFloat))
	}
	if newdata.Speed != "" {
		convFloat, _ := strconv.ParseFloat(newdata.Speed, 32)
		postingString.WriteString(fmt.Sprintf("speed=%f,", convFloat))
	}
	if newdata.Climb != "" {
		convFloat, _ := strconv.ParseFloat(newdata.Climb, 32)
		postingString.WriteString(fmt.Sprintf("climb=%f,", convFloat))
	}
	if newdata.Time == "" {
		newdata.Time = time.Now().Format("2006-01-02 15:04:05.999")
	}
	if newdata.EPV != "" {
		postingString.WriteString(fmt.Sprintf("EPV=%s,", newdata.EPV))
	}
	if newdata.EPT != "" {
		postingString.WriteString(fmt.Sprintf("EPT=%s,", newdata.EPT))
	}
	if newdata.Course != "" {
		convFloat, _ := strconv.ParseFloat(newdata.Course, 32)
		postingString.WriteString(fmt.Sprintf("Course=%f,", convFloat))
	}
	loc.Mutex.Unlock()

	return postingString
}

func (loc *Location) processTimezone() {

}
