package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/internal/core"
	"github.com/qcasey/MDroid-Core/internal/core/sessions"
	"github.com/qcasey/MDroid-Core/pkg/mserial"
	"github.com/qcasey/MDroid-Core/routes/session"
	"github.com/qcasey/MDroid-Core/routes/settings"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// MDroidRoute holds information for our meta /routes output
type MDroidRoute struct {
	Path    string `json:"Path"`
	Methods string `json:"Methods"`
}

var routes []MDroidRoute

// **
// Start with some router functions
// **

// Stop MDroid-Core service
func stopMDroid(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Stopping MDroid Service as per request")
	core.WriteNewResponse(&w, r, core.JSONResponse{Output: "OK", OK: true})
	os.Exit(0)
}

func handleSleepMDroid(w http.ResponseWriter, r *http.Request) {
	/*params := mux.Vars(r)
	msToSleepString, ok := params["millis"]
	if !ok {
		core.WriteNewResponse(&w, r, core.JSONResponse{Output: "Time to sleep required", OK: false})
		return
	}

	msToSleep, err := strconv.ParseInt(msToSleepString, 10, 64)
	if err != nil {
		core.WriteNewResponse(&w, r, core.JSONResponse{Output: "Invalid time to sleep", OK: false})
		return
	}
	*/
	core.WriteNewResponse(&w, r, core.JSONResponse{Output: "OK", OK: true})
	sleepMDroid()
}

func sleepMDroid() {
	log.Info().Msg("Going to sleep now! Powering down.")
	go func() { mserial.PushText(fmt.Sprintf("putToSleep%d", -1)) }()
	//sendServiceCommand("MDROID", "shutdown")
}

// Reset network entirely
func resetNetwork() {
	cmd := exec.Command("/etc/init.d/network", "restart")
	log.Info().Msg("Restarting network...")
	err := cmd.Run()
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}
	log.Info().Msg("Network reset complete.")
}

// Shutdown the current machine
func handleShutdown(w http.ResponseWriter, r *http.Request) {
	/*params := mux.Vars(r)
	machine, ok := params["machine"]

		if !ok {
			core.WriteNewResponse(&w, r, core.JSONResponse{Output: "Machine name required", OK: false})
			return
		}*/

	core.WriteNewResponse(&w, r, core.JSONResponse{Output: "OK", OK: true})
	/*err := sendServiceCommand(machine, "shutdown")
	if err != nil {
		log.Error().Msg(err.Error())
	}*/
}

func handleSlackAlert(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	err := sessions.SlackAlert(params["message"])
	if err != nil {
		core.WriteNewResponse(&w, r, core.JSONResponse{Output: err.Error(), OK: false})
		return
	}
	core.WriteNewResponse(&w, r, core.JSONResponse{Output: params["message"], OK: true})
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

func changeLogLevel(level zerolog.Level) {
	zerolog.SetGlobalLevel(level)
}

// **
// end router functions
// **

// SetDefaultRoutes initializes an MDroid router with default system routes
func SetDefaultRoutes(router *mux.Router) {
	log.Info().Msg("Configuring default routes...")

	//
	// Main routes
	//
	router.HandleFunc("/routes", func(w http.ResponseWriter, r *http.Request) {
		core.WriteNewResponse(&w, r, core.JSONResponse{Output: routes, OK: true})
	}).Methods("GET")
	router.HandleFunc("/shutdown/{machine}", handleShutdown).Methods("GET")
	router.HandleFunc("/{machine}/shutdown", handleShutdown).Methods("GET")
	router.HandleFunc("/stop", stopMDroid).Methods("GET")
	router.HandleFunc("/sleep", handleSleepMDroid).Methods("GET")
	router.HandleFunc("/shutdown", handleSleepMDroid).Methods("GET")
	router.HandleFunc("/alert/{message}", handleSlackAlert).Methods("GET")
	router.HandleFunc("/debug/level/{level}", handleChangeLogLevel).Methods("GET")

	//
	// Session routes
	//
	router.HandleFunc("/session", session.HandleGetAll).Methods("GET")
	router.HandleFunc("/session/{name}", session.HandleGet).Methods("GET")
	router.HandleFunc("/session/{name}", session.HandleSet).Methods("POST")

	//
	// Settings routes
	//
	router.HandleFunc("/settings", settings.HandleGetAll).Methods("GET")
	router.HandleFunc("/settings/{key}", settings.HandleGet).Methods("GET")
	router.HandleFunc("/settings/{key}/{value}", settings.HandleSet).Methods("POST")

	//
	// Finally, welcome and meta routes
	//
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		core.WriteNewResponse(&w, r, core.JSONResponse{Output: "Welcome to MDroid! This port is fully operational, see the docs or /routes for applicable routes.", OK: true})
	}).Methods("GET")
}

// Start configures default MDroid routes, starts router with optional middleware if configured
func Start(router *mux.Router) {
	// Walk routes
	err := router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		var newroute MDroidRoute

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
		err := http.ListenAndServe(":5353", router)
		log.Error().Msg(err.Error())
		log.Error().Msg("Router failed! We messed up really bad to get this far. Restarting the router...")
		time.Sleep(time.Second * 10)
	}
}
