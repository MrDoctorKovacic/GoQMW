package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/bluetooth"
	"github.com/MrDoctorKovacic/MDroid-Core/format"
	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
	"github.com/MrDoctorKovacic/MDroid-Core/pybus"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions/gps"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions/stat"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// **
// Start with some router functions
// **

// Stop MDroid-Core service
func stopMDroid(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Stopping MDroid Service as per request")
	format.WriteResponse(&w, r, format.JSONResponse{Output: "OK", OK: true})
	os.Exit(0)
}

// Reboot the machine
func handleReboot(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	machine, ok := params["machine"]

	if !ok {
		format.WriteResponse(&w, r, format.JSONResponse{Output: "Machine name required", OK: false})
		return
	}

	format.WriteResponse(&w, r, format.JSONResponse{Output: "OK", OK: true})
	sendServiceCommand(format.Name(machine), "reboot")
}

// Shutdown the current machine
func handleShutdown(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	machine, ok := params["machine"]

	if !ok {
		format.WriteResponse(&w, r, format.JSONResponse{Output: "Machine name required", OK: false})
		return
	}

	format.WriteResponse(&w, r, format.JSONResponse{Output: "OK", OK: true})
	sendServiceCommand(machine, "shutdown")
}

func handleSlackAlert(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	if settings.SlackURL != "" {
		sessions.SlackAlert(settings.SlackURL, params["message"])
		format.WriteResponse(&w, r, format.JSONResponse{Output: params["message"], OK: true})
	} else {
		format.WriteResponse(&w, r, format.JSONResponse{Output: "Slack URL not set in config.", OK: false})
	}
}

// **
// end router functions
// **

// settings.Configures routes, starts router with optional middleware if configured
func startRouter() {
	// Init router
	router := mux.NewRouter()

	//
	// Main routes
	//
	router.HandleFunc("/restart/{machine}", handleReboot).Methods("GET")
	router.HandleFunc("/shutdown/{machine}", handleShutdown).Methods("GET")
	router.HandleFunc("/{machine}/reboot", handleReboot).Methods("GET")
	router.HandleFunc("/{machine}/shutdown", handleShutdown).Methods("GET")
	router.HandleFunc("/stop", stopMDroid).Methods("GET")
	router.HandleFunc("/alert/{message}", handleSlackAlert).Methods("GET")
	router.HandleFunc("/stats", format.HandleGetStats).Methods("GET")

	//
	// GPS Routes
	//
	router.HandleFunc("/session/gps", gps.HandleGet).Methods("GET")
	router.HandleFunc("/session/gps", gps.HandleSet).Methods("POST")
	router.HandleFunc("/session/timezone", func(w http.ResponseWriter, r *http.Request) {
		response := format.JSONResponse{Output: gps.GetTimezone(), OK: true}
		format.WriteResponse(&w, r, response)
	}).Methods("GET")

	//
	// Session routes
	//
	router.HandleFunc("/session", sessions.HandleGetAll).Methods("GET")
	router.HandleFunc("/session/socket", sessions.GetSessionSocket).Methods("GET")
	router.HandleFunc("/session/{name}", sessions.HandleGet).Methods("GET")
	router.HandleFunc("/session/{name}/{checksum}", sessions.HandleSet).Methods("POST")
	router.HandleFunc("/session/{name}", sessions.HandleSet).Methods("POST")

	router.HandleFunc("/stat/", stat.HandleGetAll).Methods("GET")
	router.HandleFunc("/stat/{name}", stat.HandleGet).Methods("GET")
	router.HandleFunc("/stat/{name}", stat.HandleSet).Methods("POST")

	//
	// Settings routes
	//
	router.HandleFunc("/settings", settings.HandleGetAll).Methods("GET")
	router.HandleFunc("/settings/{component}", settings.HandleGet).Methods("GET")
	router.HandleFunc("/settings/{component}/{name}", settings.HandleGetValue).Methods("GET")
	router.HandleFunc("/settings/{component}/{name}/{value}/{checksum}", settings.HandleSet).Methods("POST")
	router.HandleFunc("/settings/{component}/{name}/{value}", settings.HandleSet).Methods("POST")

	//
	// PyBus Routes
	//
	router.HandleFunc("/pybus/{src}/{dest}/{data}/{checksum}", pybus.StartRoutine).Methods("POST")
	router.HandleFunc("/pybus/{src}/{dest}/{data}", pybus.StartRoutine).Methods("POST")
	router.HandleFunc("/pybus/{command}/{checksum}", pybus.StartRoutine).Methods("GET")
	router.HandleFunc("/pybus/{command}", pybus.StartRoutine).Methods("GET")

	//
	// Serial routes
	//
	router.HandleFunc("/serial/{command}/{checksum}", mserial.WriteSerialHandler).Methods("POST")
	router.HandleFunc("/serial/{command}", mserial.WriteSerialHandler).Methods("POST")

	//
	// Bluetooth routes
	//
	router.HandleFunc("/bluetooth", bluetooth.GetDeviceInfo).Methods("GET")
	router.HandleFunc("/bluetooth/getDeviceInfo", bluetooth.GetDeviceInfo).Methods("GET")
	router.HandleFunc("/bluetooth/getMediaInfo", bluetooth.GetMediaInfo).Methods("GET")
	router.HandleFunc("/bluetooth/disconnect", bluetooth.HandleDisconnect).Methods("GET")
	router.HandleFunc("/bluetooth/connect", bluetooth.Connect).Methods("GET")
	router.HandleFunc("/bluetooth/prev", bluetooth.Prev).Methods("GET")
	router.HandleFunc("/bluetooth/next", bluetooth.Next).Methods("GET")
	router.HandleFunc("/bluetooth/pause", bluetooth.Pause).Methods("GET")
	router.HandleFunc("/bluetooth/play", bluetooth.Play).Methods("GET")
	router.HandleFunc("/bluetooth/refresh", bluetooth.ForceRefresh).Methods("GET")

	//
	// Catch-Alls for (hopefully) a pre-approved pybus function
	// i.e. /doors/lock
	//
	router.HandleFunc("/{device}/{command}", pybus.ParseCommand).Methods("GET")

	//
	// Finally, welcome and meta routes
	//
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response := format.JSONResponse{Output: "Welcome to MDroid! This port is fully operational, see the docs for applicable routes.", OK: true}
		format.WriteResponse(&w, r, response)
	}).Methods("GET")

	// Setup checksum middleware
	router.Use(checksumMiddleware)

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
		if r.Method == "POST" {
			params := mux.Vars(r)
			checksum, ok := params["checksum"]

			if ok && checksum != "" {
				body, err := ioutil.ReadAll(r.Body)
				defer r.Body.Close() //  must close
				if err != nil {
					log.Error().Msg(fmt.Sprintf("Error reading body: %v", err))
					format.WriteResponse(&w, r, format.JSONResponse{Output: "Can't read body", OK: false})
					return
				}

				if md5.Sum(body) != md5.Sum([]byte(checksum)) {
					log.Error().Msg(fmt.Sprintf("Invalid checksum %s", checksum))
					format.WriteResponse(&w, r, format.JSONResponse{Output: fmt.Sprintf("Invalid checksum %s", checksum), OK: false})
					return
				}
				r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}
