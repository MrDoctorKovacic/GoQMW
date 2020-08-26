package hooks

import (
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type hook struct {
	key          string
	functions    []func(newValue interface{})
	lastRan      time.Time
	throttleTime time.Duration
}

// HookList will fire an event when a specific key is added to a viper instance
type HookList struct {
	hooks []hook
	lock  sync.Mutex
}

// RegisterHook adds a new hook, watching for key (or all components if name is "")
func (hl *HookList) RegisterHook(key string, throttle time.Duration, functions ...func(newValue interface{})) {
	log.Info().Msgf("Adding new hook for %s", key)
	hl.lock.Lock()
	defer hl.lock.Unlock()
	hl.hooks = append(hl.hooks, hook{key: strings.ToLower(key), throttleTime: throttle, functions: functions})
}

// RegisterHooks takes a list of componentNames to apply the same hook to
func (hl *HookList) RegisterHooks(componentNames *[]string, throttle time.Duration, functions ...func(newValue interface{})) {
	for _, name := range *componentNames {
		hl.RegisterHook(name, throttle, functions...)
	}
}

// RunHooks all hooks registered with a specific component name
func (hl *HookList) RunHooks(key string, value interface{}) {
	hl.lock.Lock()
	defer hl.lock.Unlock()

	if len(hl.hooks) == 0 {
		// No hooks registered
		return
	}

	for _, h := range hl.hooks {
		if h.key != strings.ToLower(key) {
			continue
		}

		// Ensure enough time has passed inbetween hook calls, if a throttleTime is set
		if h.throttleTime == -1 || time.Now().Sub(h.lastRan) > h.throttleTime {
			h.lastRan = time.Now()
			for _, f := range h.functions {
				go f(value)
			}
		}
	}
}
