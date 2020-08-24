package hooks

import (
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

type hook struct {
	key      string
	function func()
}

// HookList will fire an event when a specific key is added to a viper instance
type HookList struct {
	hooks []hook
	lock  sync.Mutex
}

// RegisterHook adds a new hook, watching for key (or all components if name is "")
func (hl *HookList) RegisterHook(key string, function func()) {
	log.Info().Msgf("Adding new hook for %s", key)
	hl.lock.Lock()
	defer hl.lock.Unlock()
	hl.hooks = append(hl.hooks, hook{key: strings.ToLower(key), function: function})
}

// RegisterHooks takes a list of componentNames to apply the same hook to
func (hl *HookList) RegisterHooks(componentNames *[]string, hook func()) {
	for _, name := range *componentNames {
		hl.RegisterHook(name, hook)
	}
}

// Length returns the size of the hook list
func (hl *HookList) Length() int {
	return len(hl.hooks)
}

// RunHooks all hooks registered with a specific component name
func (hl *HookList) RunHooks(key string) {
	hl.lock.Lock()
	defer hl.lock.Unlock()

	log.Info().Msgf("Running hooks for key %s", key)

	if len(hl.hooks) == 0 {
		// No hooks registered
		return
	}

	for _, h := range hl.hooks {
		if h.key == strings.ToLower(key) || h.key == "" {
			go h.function()
		}
	}
}
