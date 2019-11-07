package sessions

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/rs/zerolog/log"
)

// Data holds the data and last update info for each session value
type Data struct {
	Name       string `json:"name,omitempty"`
	Value      string `json:"value,omitempty"`
	LastUpdate string `json:"lastUpdate,omitempty"`
	Quiet      bool   `json:"quiet,omitempty"`
}

// Measurement is struct for session values stored as float32
type Measurement struct {
	Name       string `json:"name,omitempty"`
	Value      string `json:"value,omitempty"`
	LastUpdate string `json:"lastUpdate,omitempty"`
	Quiet      bool   `json:"quiet,omitempty"`
}

// Session is a mapping of Datas, which contain session values
type Session struct {
	data         map[string]Data
	measurements map[string]Measurement
	Mutex        sync.Mutex
	file         string
}

type hooks struct {
	list  map[string][]func(triggerPackage *Data)
	count int
	mutex sync.Mutex
}

var hookList hooks
var session Session

func init() {
	session.data = make(map[string]Data)
	hookList = hooks{list: make(map[string][]func(triggerPackage *Data), 0), count: 0}
}

// RegisterHook adds a new hook into a settings change
func RegisterHook(componentName string, hook func(triggerPackage *Data)) {
	log.Info().Msg(fmt.Sprintf("Adding new hook for %s", componentName))
	hookList.mutex.Lock()
	defer hookList.mutex.Unlock()
	hookList.list[componentName] = append(hookList.list[componentName], hook)
	hookList.count++
}

// RegisterHookSlice takes a list of componentNames to apply the same hook to
func RegisterHookSlice(componentNames *[]string, hook func(triggerPackage *Data)) {
	for _, name := range *componentNames {
		RegisterHook(name, hook)
	}
}

// Runs all hooks registered with a specific component name
func runHooks(triggerPackage Data) {
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

	log.Info().Msg(fmt.Sprintf("response Status: %s", resp.Status))
	log.Info().Msg(fmt.Sprintf("response Headers: %s", resp.Header))

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Info().Msg(fmt.Sprintf("response Body: %s", string(body)))
	return nil
}
