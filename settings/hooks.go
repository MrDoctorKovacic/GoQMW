package settings

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
)

type hooks struct {
	list  map[string][]func(settingName string, settingValue string)
	count int
	mutex sync.Mutex
}

var hookList hooks

func init() {
	hookList = hooks{list: make(map[string][]func(settingName string, settingValue string), 0), count: 0}
}

// RegisterHook adds a new hook into a settings change
func RegisterHook(componentName string, hook func(settingName string, settingValue string)) {
	log.Info().Msg(fmt.Sprintf("Adding new hook for %s", componentName))
	hookList.mutex.Lock()
	defer hookList.mutex.Unlock()
	hookList.list[componentName] = append(hookList.list[componentName], hook)
	hookList.count++
}

// Runs all hooks registered with a specific component name
func runHooks(componentName string, settingName string, settingValue string) {
	hookList.mutex.Lock()
	defer hookList.mutex.Unlock()
	allHooks, ok := hookList.list[componentName]

	if !ok || len(allHooks) == 0 {
		// No hooks registered for component
		return
	}

	for _, h := range allHooks {
		go h(settingName, settingValue)
	}
}
