package sessions

import (
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

type hook struct {
	key      string
	function func()
}

var hookList []hook
var hookLock sync.Mutex

func init() {
}

// RegisterHook adds a new hook, watching for key (or all components if name is "")
func RegisterHook(key string, function func()) {
	log.Info().Msgf("Adding new hook for %s", key)
	hookLock.Lock()
	defer hookLock.Unlock()
	hookList = append(hookList, hook{key: strings.ToLower(key), function: function})
}

// RegisterHooks takes a list of componentNames to apply the same hook to
func RegisterHooks(componentNames *[]string, hook func()) {
	for _, name := range *componentNames {
		RegisterHook(name, hook)
	}
}

// Runs all hooks registered with a specific component name
func runHooks(key string) {
	hookLock.Lock()
	defer hookLock.Unlock()

	if len(hookList) == 0 {
		// No hooks registered
		return
	}

	for _, h := range hookList {
		if h.key == strings.ToLower(key) || h.key == "" {
			go h.function()
		}
	}
}
