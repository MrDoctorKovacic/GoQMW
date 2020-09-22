package core

import (
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/routes/session"
	"github.com/qcasey/MDroid-Core/routes/settings"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// mDroidRoute holds information for our meta /routes output
type mDroidRoute struct {
	Path    string `json:"Path"`
	Methods string `json:"Methods"`
}

var routes []mDroidRoute

// Start configures default MDroid routes, starts router with optional middleware if configured
func (core *Core) Start() {
	// Walk routes
	err := core.Router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		var newroute mDroidRoute

		pathTemplate, err := route.GetPathTemplate()
		if err == nil {
			newroute.Path = pathTemplate
		}
		methods, err := route.GetMethods()
		if err == nil {
			newroute.Methods = strings.Join(methods, ",")
		}
		routes = append(routes, newroute)
		return nil
	})

	if err != nil {
		log.Error().Msg(err.Error())
	}

	log.Info().Msg("Starting server...")

	// Start the router in an endless loop
	for {
		err := http.ListenAndServe(":5353", core.Router)
		log.Error().Msg(err.Error())
		log.Error().Msg("Router failed! We messed up really bad to get this far. Restarting the router...")
		time.Sleep(time.Second * 10)
	}
}

func (core *Core) injectRoutes() {
	//
	// Main routes
	//
	core.Router.HandleFunc("/routes", func(w http.ResponseWriter, r *http.Request) {
		WriteNewResponse(&w, r, JSONResponse{Output: routes, OK: true})
	}).Methods("GET")
	core.Router.HandleFunc("/debug/level/{level}", handleChangeLogLevel).Methods("GET")

	//
	// Session routes
	//
	core.Router.HandleFunc("/session", session.HandleGetAll).Methods("GET")
	core.Router.HandleFunc("/session/{name}", session.HandleGet).Methods("GET")
	core.Router.HandleFunc("/session/{name}", session.HandleSet).Methods("POST")

	//
	// Settings routes
	//
	core.Router.HandleFunc("/settings", settings.HandleGetAll).Methods("GET")
	core.Router.HandleFunc("/settings/{key}", settings.HandleGet).Methods("GET")
	core.Router.HandleFunc("/settings/{key}/{value}", settings.HandleSet).Methods("POST")

	//
	// Finally, welcome and meta routes
	//
	core.Router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		WriteNewResponse(&w, r, JSONResponse{Output: "Welcome to MDroid! This port is fully operational, see the docs or /routes for applicable routes.", OK: true})
	}).Methods("GET")
}

func handleChangeLogLevel(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	level := FormatName(params["level"])
	switch level {
	case "INFO":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "DEBUG":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "ERROR":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		WriteNewResponse(&w, r, JSONResponse{Output: "Invalid log level.", OK: false})
	}
	WriteNewResponse(&w, r, JSONResponse{Output: level, OK: true})
}
