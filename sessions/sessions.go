package sessions

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/rs/zerolog/log"
)

// Value holds the data and last update info for each session value
type Value struct {
	Value      string `json:"value,omitempty"`
	LastUpdate string `json:"lastUpdate,omitempty"`
	Quiet      bool   `json:"quiet,omitempty"`
}

// SessionPackage contains both name and data
type SessionPackage struct {
	Name string
	Data Value
}

// Session is a mapping of SessionPackages, which contain session values
type Session struct {
	data  map[string]Value
	Mutex sync.Mutex
	file  string
}

type hooks struct {
	list  map[string][]func(triggerPackage *SessionPackage)
	count int
	mutex sync.Mutex
}

var hookList hooks
var session Session

func init() {
	session.data = make(map[string]Value)
	hookList = hooks{list: make(map[string][]func(triggerPackage *SessionPackage), 0), count: 0}
}

// RegisterHook adds a new hook into a settings change
func RegisterHook(componentName string, hook func(triggerPackage *SessionPackage)) {
	log.Info().Msg(fmt.Sprintf("Adding new hook for %s", componentName))
	hookList.mutex.Lock()
	defer hookList.mutex.Unlock()
	hookList.list[componentName] = append(hookList.list[componentName], hook)
	hookList.count++
}

// RegisterHookSlice takes a list of componentNames to apply the same hook to
func RegisterHookSlice(componentNames *[]string, hook func(triggerPackage *SessionPackage)) {
	for _, name := range *componentNames {
		RegisterHook(name, hook)
	}
}

// Runs all hooks registered with a specific component name
func runHooks(triggerPackage SessionPackage) {
	hookList.mutex.Lock()
	defer hookList.mutex.Unlock()
	allHooks, ok := hookList.list[triggerPackage.Name]

	if !ok || len(allHooks) == 0 {
		// No hooks registered for component
		return
	}

	for _, h := range allHooks {
		go h(&triggerPackage)
	}
}

// SlackAlert sends a message to a slack channel webhook
func SlackAlert(channel string, message string) {
	if channel == "" {
		log.Warn().Msg("Empty slack channel")
		return
	}
	if message == "" {
		log.Warn().Msg("Empty slack message")
		return
	}

	jsonStr := []byte(fmt.Sprintf(`{"text":"%s"}`, message))
	req, err := http.NewRequest("POST", channel, bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	log.Info().Msg(fmt.Sprintf("response Status: %s", resp.Status))
	log.Info().Msg(fmt.Sprintf("response Headers: %s", resp.Header))

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}
	log.Info().Msg(fmt.Sprintf("response Body: %s", string(body)))
}
