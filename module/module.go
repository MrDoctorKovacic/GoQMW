package module

import "github.com/gorilla/mux"

// Module handles interconnects / interfaces with
// MDroid core (sessions, router, etc) and the rest of the system
type Module interface {
	Setup(configAddr *map[string]string)
	SetRoutes(router mux.Router)
}
