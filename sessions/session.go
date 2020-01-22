package sessions

import (
	"bytes"
	"container/list"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/qcasey/MDroid-Core/format/response"
	"github.com/qcasey/MDroid-Core/settings"
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
	dataSample       *list.List
	throughput       float64
	ThroughputString string `json:"Throughput"`
	Sets             uint32 `json:"Sets"`
	Gets             uint32 `json:"Gets"`
	DipsBelowMinimum int    `json:"DipsBelowMinimum"`
}

// Session is a mapping of Datas, which contain session values
type Session struct {
	data              map[string]Data
	stats             Stats
	Mutex             sync.RWMutex
	file              string
	startTime         time.Time
	throughputWarning int
}

var session Session

func init() {
	session.data = make(map[string]Data)
	session.stats.dataSample = list.New()
	session.startTime = time.Now()
	session.throughputWarning = -1
}

// Setup prepares valid tokens from settings file
func Setup(configAddr *map[string]string) {
	configMap := *configAddr

	InitializeDefaults()

	// Set up Auth tokens
	token, usingTokens := configMap["AUTH_TOKEN"]
	serverHost, usingCentralHost := configMap["MDROID_SERVER"]
	if !usingTokens || !usingCentralHost {
		log.Warn().Msg("Missing central host parameters - checking into central host has been disabled. Are you sure this is correct?")
	} else {
		log.Info().Msg("Successfully set up auth tokens")
	}

	// Setup throughput warnings
	throughputString, usingThroughputCheck := configMap["THROUGHPUT_WARN_THRESHOLD"]
	if usingThroughputCheck {
		throughput, err := strconv.Atoi(throughputString)
		if err == nil {
			session.throughputWarning = throughput
		}
	}

	go checkServer(serverHost, token)
}

// InitializeDefaults sets default session values here
func InitializeDefaults() {
	SetValue("VIDEO_ON", "TRUE")
}

// GetStartTime will give the time the session started
func GetStartTime() time.Time {
	return session.startTime
}

// HandleGetStats will return various statistics on this Session
func HandleGetStats(w http.ResponseWriter, r *http.Request) {
	session.Mutex.RLock()
	defer session.Mutex.RUnlock()
	session.stats.calcThroughput()

	response.WriteNew(&w, r, response.JSONResponse{Output: session.stats, OK: true})
}

func (s *Stats) calcThroughput() {
	d := session.stats.dataSample.Front()
	data := d.Value.(Data)
	s.throughput = float64(session.stats.dataSample.Len()) / time.Since(data.date).Seconds()
	s.ThroughputString = fmt.Sprintf("%f sets per second", s.throughput)

	if session.throughputWarning >= 0 && session.stats.throughput < float64(session.throughputWarning) {
		session.stats.DipsBelowMinimum++
		SlackAlert("Throughput has fallen below 20 sets/second")
	}
}

func addStat(d Data) {
	session.stats.dataSample.PushBack(d)
	if session.stats.dataSample.Len() > 300 {
		session.stats.dataSample.Remove(session.stats.dataSample.Front())
	}

	// Check throughput every so often
	if session.stats.Sets%500 == 0 {
		session.stats.calcThroughput()
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
