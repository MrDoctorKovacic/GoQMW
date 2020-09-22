package sessions

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/qcasey/MDroid-Core/hooks"
	"github.com/qcasey/MDroid-Core/internal/core/settings"
	"github.com/qcasey/MDroid-Core/pkg/db"
	"github.com/qcasey/MDroid-Core/pkg/mqtt"
	"github.com/qcasey/viper"
	"github.com/rs/zerolog/log"
)

// Package holds the Package and last update info for each session value
type Package struct {
	Name       string      `json:"name,omitempty"`
	Value      interface{} `json:"value,omitempty"`
	LastUpdate string      `json:"lastUpdate,omitempty"`
	date       time.Time
	Quiet      bool `json:"quiet,omitempty"`
}

// Session is a mapping of Datas, which contain session values
type Session struct {
	Mutex             sync.RWMutex
	file              string
	startTime         time.Time
	throughputWarning int
}

var session Session

// HL holds the registered hooks for settings
var HL *hooks.HookList

// Data holds a viper instance
var Data *viper.Viper

func init() {
	Data = viper.New()
	session.startTime = time.Now()
	session.throughputWarning = -1
	HL = new(hooks.HookList)
}

// Setup prepares valid tokens from settings file
func Setup() {
	// Setup throughput warnings
	if settings.Data.IsSet("mdroid.THROUGHPUT_WARN_THRESHOLD") {
		session.throughputWarning = settings.Data.GetInt("THROUGHPUT_WARN_THRESHOLD")
	}
}

// GetStartTime will give the time the session started
func GetStartTime() time.Time {
	return session.startTime
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

// Set prepares a Value structure before passing it to the setter
func Set(key string, value interface{}, publishToRemote bool) string {
	keyAlreadyExists := Data.IsSet(key)

	addToSession(key, value)

	// Insert into database if this is a new/updated value
	if !keyAlreadyExists || Data.Get(fmt.Sprintf("%s.value", key)) != value {
		formattedName := strings.ToLower(strings.Replace(strings.Replace(key, " ", "_", -1), ".", "/", -1))

		topic := fmt.Sprintf("session/%s", formattedName)
		go mqtt.Publish(topic, fmt.Sprintf("%v", value), publishToRemote)

		if db.DB != nil {
			// Convert to a float if that suits the value, otherwise change field to value_string
			valueString := fmt.Sprintf("value=%s", value)
			/*if _, err := strconv.ParseFloat(value, 32); err != nil {
				valueString = fmt.Sprintf("value=\"%s\"", value)
			}*/

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

func addToSession(key string, value interface{}) {
	oldKeyWrites := Data.GetInt(fmt.Sprintf("%s.writes", key))

	Data.Set(fmt.Sprintf("%s.value", key), value)
	Data.Set(fmt.Sprintf("%s.write_date", key), time.Now())
	Data.Set(fmt.Sprintf("%s.writes", key), oldKeyWrites+1)

	// Finish post processing
	go HL.RunHooks(key, value)
}
