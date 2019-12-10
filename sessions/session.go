package sessions

import (
	"bytes"
	"container/list"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/format"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/rs/zerolog/log"
)

// Data holds the data and last update info for each session value
type Data struct {
	Name       string `json:"name,omitempty"`
	Value      string `json:"value,omitempty"`
	LastUpdate string `json:"lastUpdate,omitempty"`
	date       time.Time
	Quiet      bool `json:"quiet,omitempty"`
}

// Stats hold simple metrics for the session as a whole
type Stats struct {
	dataSample *list.List
	Throughput string `json:"Throughput"`
	Sets       int    `json:"Sets"`
	Gets       int    `json:"Gets"`
}

// Session is a mapping of Datas, which contain session values
type Session struct {
	data  map[string]Data
	stats Stats
	Mutex sync.RWMutex
	file  string
}

var session Session

func init() {
	session.data = make(map[string]Data)
	session.stats.dataSample = list.New()

}

// InitializeDefaults sets default session values here
func InitializeDefaults() {
	SetValue("VIDEO_ON", "TRUE")
}

// HandleGetStats will return various statistics on this Session
func HandleGetStats(w http.ResponseWriter, r *http.Request) {
	session.Mutex.RLock()
	defer session.Mutex.RUnlock()
	d := session.stats.dataSample.Front()
	data := d.Value.(Data)
	session.stats.Throughput = fmt.Sprintf("%f sets per second", float64(session.stats.dataSample.Len())/time.Since(data.date).Seconds())

	format.WriteResponse(&w, r, format.JSONResponse{Output: session.stats, OK: true})
}

func addStat(d Data) {
	session.stats.dataSample.PushBack(d)
	if session.stats.dataSample.Len() > 300 {
		session.stats.dataSample.Remove(session.stats.dataSample.Front())
	}
}

// SlackAlert sends a message to a slack channel webhook
func SlackAlert(message string) error {
	channel, err := settings.Get("MDROID", "SLACK_URL")
	if err != nil || channel == "" {
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
