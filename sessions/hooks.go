package sessions

//
// This file contains modifier functions for the main session defined in session.go
// These take a POSTed value and start triggers or make adjustments
//
// Most here are specific to my setup only, and likely not generalized
//

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
)

type hooks struct {
	list  map[string][]func(triggerPackage *SessionPackage)
	count int
	mutex sync.Mutex
}

var hookList hooks

func init() {
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
