package sessions

import (
	"bytes"
	"container/list"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/db"
	"github.com/qcasey/MDroid-Core/format/response"
	"github.com/qcasey/MDroid-Core/gps"
	"github.com/qcasey/MDroid-Core/mqtt"
	"github.com/qcasey/MDroid-Core/settings"
	"github.com/qcasey/viper"
	"github.com/rs/zerolog/log"
)

// Package holds the Package and last update info for each session value
type Package struct {
	Name       string `json:"name,omitempty"`
	Value      string `json:"value,omitempty"`
	LastUpdate string `json:"lastUpdate,omitempty"`
	date       time.Time
	Quiet      bool `json:"quiet,omitempty"`
}

// Stats hold simple metrics for the session as a whole
type Stats struct {
	dataSample       *list.List
	throughput       float64
	ThroughputString string `json:"Throughput"`
	Sets             uint32 `json:"Sets"`
	Gets             uint32 `json:"Gets"`
	DipsBelowMinimum int    `json:"DipsBelowMinimum"`
}

// Session is a mapping of Datas, which contain session values
type Session struct {
	stats             Stats
	Mutex             sync.RWMutex
	file              string
	startTime         time.Time
	throughputWarning int
}

var session Session

// Data holds a viper instance
var Data *viper.Viper

func init() {
	Data = viper.New()
	session.stats.dataSample = list.New()
	session.startTime = time.Now()
	session.throughputWarning = -1
}

// Setup prepares valid tokens from settings file
func Setup() {
	// Setup throughput warnings
	if settings.Data.IsSet("mdroid.THROUGHPUT_WARN_THRESHOLD") {
		session.throughputWarning = settings.Data.GetInt("THROUGHPUT_WARN_THRESHOLD")
	}
}

// HandleGet returns a specific session value
func HandleGet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	sessionValue := Data.Get(params["name"])
	response := response.JSONResponse{Output: sessionValue, OK: true}
	if !Data.IsSet(params["name"]) {
		response.Output = "Does not exist"
		response.OK = false
	}
	response.Write(&w, r)
}

// HandleGetAll responds to an HTTP request for the entire session
func HandleGetAll(w http.ResponseWriter, r *http.Request) {
	//requestingMin := r.URL.Query().Get("min") == "1"
	response := response.JSONResponse{OK: true}
	response.Output = Data.AllSettings()
	response.Write(&w, r)
}

// GetStartTime will give the time the session started
func GetStartTime() time.Time {
	return session.startTime
}

// HandleGetStats will return various statistics on this Session
func HandleGetStats(w http.ResponseWriter, r *http.Request) {
	session.Mutex.RLock()
	defer session.Mutex.RUnlock()
	session.stats.calculateThroughput()

	response.WriteNew(&w, r, response.JSONResponse{Output: session.stats, OK: true})
}

func (s *Stats) calculateThroughput() {
	d := session.stats.dataSample.Front()
	data := d.Value.(Package)
	s.throughput = float64(session.stats.dataSample.Len()) / time.Since(data.date).Seconds()
	s.ThroughputString = fmt.Sprintf("%f sets per second", s.throughput)

	if session.throughputWarning >= 0 && session.stats.throughput < float64(session.throughputWarning) {
		session.stats.DipsBelowMinimum++
		SlackAlert("Throughput has fallen below 20 sets/second")
	}
}

func addStat(d Package) {
	session.stats.dataSample.PushBack(d)
	if session.stats.dataSample.Len() > 300 {
		session.stats.dataSample.Remove(session.stats.dataSample.Front())
	}

	// Check throughput every so often
	if session.stats.Sets%500 == 0 {
		session.stats.calculateThroughput()
	}
}

// SlackAlert sends a message to a slack channel webhook
func SlackAlert(message string) error {
	channel := settings.Data.GetString("MDROID.SLACK_URL")
	if channel == "" {
		return fmt.Errorf("Empty slack channel")
	}
	if message == "" {
		return fmt.Errorf("Empty slack message")
	}

	jsonStr := []byte(fmt.Sprintf(`{"text":"%s"}`, message))
	req, err := http.NewRequest("POST", channel, bytes.NewBuffer(jsonStr))
	if err != nil {
		return err
	}
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	log.Info().Msgf("response Status: %s", resp.Status)
	log.Info().Msgf("response Headers: %s", resp.Header)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Info().Msgf("response Body: %s", string(body))
	return nil
}

// HandleSet updates or posts a new session value to the common session
func HandleSet(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)

	// Default to NOT OK response
	response := response.JSONResponse{OK: false}

	if err != nil {
		log.Error().Msgf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	// Put body back
	r.Body.Close() //  must close
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	if len(body) == 0 {
		response.Output = "Error: Empty body"
		response.Write(&w, r)
		return
	}

	params := mux.Vars(r)
	var newdata Package

	if err = json.NewDecoder(r.Body).Decode(&newdata); err != nil {
		log.Error().Msgf("Error decoding incoming JSON:\n%s", err.Error())
		response.Output = err.Error()
		response.Write(&w, r)
		return
	}

	// Call the setter
	newdata.Name = params["name"]
	Set(params["name"], newdata.Value)

	// Craft OK response
	response.OK = true
	response.Output = newdata

	response.Write(&w, r)
}

// Set prepares a Value structure before passing it to the setter
func Set(key string, value interface{}) string {
	keyAlreadyExists := Data.IsSet(key)
	oldKeyValue := Data.Get(key)

	Data.Set(key, value)
	Data.Set(fmt.Sprintf("%s_meta.lastUpdate", key), time.Now().In(gps.GetTimezone()))
	session.stats.Sets++

	// Finish post processing
	go runHooks(strings.ToLower(key))

	// Insert into database if this is a new/updated value
	if !keyAlreadyExists || (keyAlreadyExists && oldKeyValue != value) {
		formattedName := strings.ToLower(strings.Replace(key, " ", "_", -1))

		topic := fmt.Sprintf("session/%s", formattedName)
		go mqtt.Publish(topic, Data.GetString(key), true)

		if db.DB != nil {
			// Convert to a float if that suits the value, otherwise change field to value_string
			valueString := fmt.Sprintf("value=%s", Data.GetString(key))
			if _, err := strconv.ParseFloat(Data.GetString(key), 32); err != nil {
				valueString = fmt.Sprintf("value=\"%s\"", Data.GetString(key))
			}

			// In Sessions, all values come in and out as strings regardless,
			// but this conversion alows Influx queries on the floats to be executed
			err := db.DB.Write(fmt.Sprintf("%s %s", formattedName, valueString))
			if err != nil {
				errorText := fmt.Sprintf("Error writing %s to database:\n%s", valueString, err.Error())
				// Only spam our log if Influx is online
				if db.DB.Started {
					log.Error().Msg(errorText)
				}
				return fmt.Sprintf(errorText)
			}
		}
	}

	return key
}
