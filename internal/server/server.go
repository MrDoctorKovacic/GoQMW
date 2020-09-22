package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/internal/core"
	"github.com/qcasey/MDroid-Core/internal/server/routes/session"
	"github.com/qcasey/MDroid-Core/internal/server/routes/settings"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Server binds the interal MDroid core and router together
type Server struct {
	Core   *core.Core
	Router *mux.Router
}

// mDroidRoute holds information for our meta /routes output
type mDroidRoute struct {
	Path    string `json:"Path"`
	Methods string `json:"Methods"`
}

var routes []mDroidRoute

// New creates a new server with underlying Core
func New(settingsFile string) *Server {
	srv := &Server{
		Router: mux.NewRouter(),
		Core:   core.New(settingsFile),
	}
	// Setup router
	srv.injectRoutes()
	return srv
}

// Start configures default MDroid routes, starts router with optional middleware if configured
func (srv *Server) Start() {
	// Walk routes
	err := srv.Router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
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
		err := http.ListenAndServe(":5353", srv.Router)
		log.Error().Msg(err.Error())
		log.Error().Msg("Router failed! We messed up really bad to get this far. Restarting the router...")
		time.Sleep(time.Second * 10)
	}
}

func (srv *Server) injectRoutes() {
	//
	// Main routes
	//
	srv.Router.HandleFunc("/routes", func(w http.ResponseWriter, r *http.Request) {
		core.WriteNewResponse(&w, r, core.JSONResponse{Output: routes, OK: true})
	}).Methods("GET")
	srv.Router.HandleFunc("/debug/level/{level}", handleChangeLogLevel).Methods("GET")

	//
	// Session routes
	//
	srv.Router.HandleFunc("/session", session.GetAll(srv.Core)).Methods("GET")
	srv.Router.HandleFunc("/session/{name}", session.Get(srv.Core)).Methods("GET")
	srv.Router.HandleFunc("/session/{name}", session.Set(srv.Core)).Methods("POST")

	//
	// Settings routes
	//
	srv.Router.HandleFunc("/settings", settings.GetAll(srv.Core)).Methods("GET")
	srv.Router.HandleFunc("/settings/{key}", settings.Get(srv.Core)).Methods("GET")
	srv.Router.HandleFunc("/settings/{key}/{value}", settings.Set(srv.Core)).Methods("POST")

	//
	// Finally, welcome and meta routes
	//
	srv.Router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		core.WriteNewResponse(&w, r, core.JSONResponse{Output: "Welcome to MDroid! This port is fully operational, see the docs or /routes for applicable routes.", OK: true})
	}).Methods("GET")
}

func handleChangeLogLevel(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	level := core.FormatName(params["level"])
	switch level {
	case "INFO":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "DEBUG":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "ERROR":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		core.WriteNewResponse(&w, r, core.JSONResponse{Output: "Invalid log level.", OK: false})
	}
	core.WriteNewResponse(&w, r, core.JSONResponse{Output: level, OK: true})
}
