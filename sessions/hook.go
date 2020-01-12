package sessions

import (
	"sync"

	"github.com/rs/zerolog/log"
)

type hook struct {
	componentName string
	function      func(triggerPackage *Data)
}

var hookList []hook
var hookLock sync.Mutex

func init() {
}

// RegisterHook adds a new hook, watching for componentName (or all components if name is "")
func RegisterHook(componentName string, function func(triggerPackage *Data)) {
	log.Info().Msgf("Adding new hook for %s", componentName)
	hookLock.Lock()
	defer hookLock.Unlock()
	hookList = append(hookList, hook{componentName: componentName, function: function})
}

// RegisterHookSlice takes a list of componentNames to apply the same hook to
func RegisterHookSlice(componentNames *[]string, hook func(triggerPackage *Data)) {
	for _, name := range *componentNames {
		RegisterHook(name, hook)
	}
}

// Runs all hooks registered with a specific component name
func runHooks(triggerPackage Data) {
	hookLock.Lock()
	defer hookLock.Unlock()

	if len(hookList) == 0 {
		// No hooks registered
		return
	}

	for _, h := range hookList {
		if h.componentName == triggerPackage.Name || h.componentName == "" {
			go h.function(&triggerPackage)
		}
	}
}
