package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/format"
	"github.com/qcasey/MDroid-Core/format/response"
	"github.com/qcasey/MDroid-Core/mserial"
	"github.com/qcasey/MDroid-Core/sessions"
	"github.com/qcasey/MDroid-Core/settings"
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
	response.WriteNew(&w, r, response.JSONResponse{Output: "OK", OK: true})
	os.Exit(0)
}

func handleSleepMDroid(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	msToSleepString, ok := params["millis"]
	if !ok {
		response.WriteNew(&w, r, response.JSONResponse{Output: "Time to sleep required", OK: false})
		return
	}

	msToSleep, err := strconv.ParseInt(msToSleepString, 10, 64)
	if err != nil {
		response.WriteNew(&w, r, response.JSONResponse{Output: "Invalid time to sleep", OK: false})
		return
	}

	response.WriteNew(&w, r, response.JSONResponse{Output: "OK", OK: true})
	sleepMDroid()
}

func sleepMDroid() {
	log.Info().Msg("Going to sleep now! Powering down.")
	go func() { mserial.PushText(fmt.Sprintf("putToSleep%d", -1)) }()
	sendServiceCommand("MDROID", "shutdown")
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

// Reboot the machine
func handleReboot(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	machine, ok := params["machine"]

	if !ok {
		response.WriteNew(&w, r, response.JSONResponse{Output: "Machine name required", OK: false})
		return
	}

	response.WriteNew(&w, r, response.JSONResponse{Output: "OK", OK: true})
	err := sendServiceCommand(format.Name(machine), "reboot")
	if err != nil {
		log.Error().Msg(err.Error())
	}
}

// Shutdown the current machine
func handleShutdown(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	machine, ok := params["machine"]

	if !ok {
		response.WriteNew(&w, r, response.JSONResponse{Output: "Machine name required", OK: false})
		return
	}

	response.WriteNew(&w, r, response.JSONResponse{Output: "OK", OK: true})
	err := sendServiceCommand(machine, "shutdown")
	if err != nil {
		log.Error().Msg(err.Error())
	}
}

// sendServiceCommand sends a command to a network machine, using a simple python server to recieve
func sendServiceCommand(name string, command string) error {
	machineServiceAddress, err := settings.Get(format.Name(name), "ADDRESS")
	if machineServiceAddress == "" {
		return fmt.Errorf("Device %s address not found, not issuing %s", name, command)
	}

	client := http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Get(fmt.Sprintf("http://%s:5350/%s", machineServiceAddress, command))
	if err != nil {
		return fmt.Errorf("Failed to command machine %s (at %s) to %s: \n%s", name, machineServiceAddress, command, err.Error())
	}

	return resp.Body.Close()
}

func handleSlackAlert(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	err := sessions.SlackAlert(params["message"])
	if err != nil {
		response.WriteNew(&w, r, response.JSONResponse{Output: err.Error(), OK: false})
		return
	}
	response.WriteNew(&w, r, response.JSONResponse{Output: params["message"], OK: true})
}

func handleChangeLogLevel(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	level := format.Name(params["level"])
	switch level {
	case "INFO":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "DEBUG":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "ERROR":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		response.WriteNew(&w, r, response.JSONResponse{Output: "Invalid log level.", OK: false})
	}
	response.WriteNew(&w, r, response.JSONResponse{Output: level, OK: true})
}

func changeLogLevel(level zerolog.Level) {
	zerolog.SetGlobalLevel(level)
}

// **
// end router functions
// **

// Start configures default MDroid routes, starts router with optional middleware if configured
func Start(router *mux.Router) {
	log.Info().Msg("Configuring default routes...")

	//
	// Main routes
	//
	router.HandleFunc("/routes", func(w http.ResponseWriter, r *http.Request) {
		response.WriteNew(&w, r, response.JSONResponse{Output: routes, OK: true})
	}).Methods("GET")
	router.HandleFunc("/restart/{machine}", handleReboot).Methods("GET")
	router.HandleFunc("/shutdown/{machine}", handleShutdown).Methods("GET")
	router.HandleFunc("/{machine}/reboot", handleReboot).Methods("GET")
	router.HandleFunc("/{machine}/shutdown", handleShutdown).Methods("GET")
	router.HandleFunc("/stop", stopMDroid).Methods("GET")
	router.HandleFunc("/sleep/{millis}", handleSleepMDroid).Methods("GET")
	router.HandleFunc("/alert/{message}", handleSlackAlert).Methods("GET")
	router.HandleFunc("/responses/stats", response.HandleGetStats).Methods("GET")
	router.HandleFunc("/debug/level/{level}", handleChangeLogLevel).Methods("GET")

	//
	// Session routes
	//
	router.HandleFunc("/session", sessions.HandleGetAll).Methods("GET")
	router.HandleFunc("/session/socket", sessions.GetSessionSocket).Methods("GET")
	router.HandleFunc("/session/stats", sessions.HandleGetStats).Methods("GET")
	router.HandleFunc("/session/{name}", sessions.HandleGet).Methods("GET")
	router.HandleFunc("/session/{name}/{checksum}", sessions.HandleSet).Methods("POST")
	router.HandleFunc("/session/{name}", sessions.HandleSet).Methods("POST")

	//
	// Settings routes
	//
	router.HandleFunc("/settings", settings.HandleGetAll).Methods("GET")
	router.HandleFunc("/settings/{component}", settings.HandleGet).Methods("GET")
	router.HandleFunc("/settings/{component}/{name}", settings.HandleGetValue).Methods("GET")
	router.HandleFunc("/settings/{component}/{name}/{value}/{checksum}", settings.HandleSet).Methods("POST")
	router.HandleFunc("/settings/{component}/{name}/{value}", settings.HandleSet).Methods("POST")

	//
	// GraphQL Implementation
	//
	router.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		result := executeQuery(r.URL.Query().Get("query"), schema)
		json.NewEncoder(w).Encode(result)
	})

	//
	// Finally, welcome and meta routes
	//
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response.WriteNew(&w, r, response.JSONResponse{Output: "Welcome to MDroid! This port is fully operational, see the docs or /routes for applicable routes.", OK: true})
	}).Methods("GET")

	// Setup checksum middleware
	router.Use(checksumMiddleware)

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

// authMiddleware will match http bearer token again the one hardcoded in our config
/*
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		reqToken := r.Header.Get("Authorization")
		splitToken := strings.Split(reqToken, "Bearer")
		if len(splitToken) != 2 || strings.TrimSpace(splitToken[1]) != settings.AuthToken {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403 - Invalid Auth Token!"))
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}*/

func checksumMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for r.Method == "POST" {
			params := mux.Vars(r)
			checksum, ok := params["checksum"]

			if !ok || checksum == "" {
				break
			}

			body, err := ioutil.ReadAll(r.Body)
			defer r.Body.Close() //  must close
			if err != nil {
				log.Error().Msgf("Error reading body: %v", err)
				response.WriteNew(&w, r, response.JSONResponse{Output: "Can't read body", OK: false})
				break
			}

			if md5.Sum(body) != md5.Sum([]byte(checksum)) {
				log.Error().Msgf("Invalid checksum %s", checksum)
				response.WriteNew(&w, r, response.JSONResponse{Output: fmt.Sprintf("Invalid checksum %s", checksum), OK: false})
				break
			}
			r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			break
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}
