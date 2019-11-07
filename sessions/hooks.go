package sessions

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

var hookList hooks

func init() {
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
