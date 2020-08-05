package settings

import (
	"sync"

	"github.com/rs/zerolog/log"
)

type hooks struct {
	list  map[string][]func(key string, value interface{})
	count int
	mutex sync.Mutex
}

var hookList hooks

func init() {
	hookList = hooks{list: make(map[string][]func(key string, value interface{}), 0), count: 0}
}

// RegisterHook adds a new hook into a settings change
func RegisterHook(key string, hook func(key string, value interface{})) {
	log.Info().Msgf("Adding new hook for %s", key)
	hookList.mutex.Lock()
	defer hookList.mutex.Unlock()
	hookList.list[key] = append(hookList.list[key], hook)
	hookList.count++
}

// Runs all hooks registered with a specific component name
func runHooks(key string, value interface{}) {
	hookList.mutex.Lock()
	defer hookList.mutex.Unlock()
	allHooks, ok := hookList.list[key]

	if !ok || len(allHooks) == 0 {
		// No hooks registered for component
		return
	}

	for _, h := range allHooks {
		go h(key, value)
	}
}
