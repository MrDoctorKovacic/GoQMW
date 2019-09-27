package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/MrDoctorKovacic/MDroid-Core/bluetooth"
	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/pybus"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/MrDoctorKovacic/MDroid-Core/gps"
	"github.com/gorilla/mux"
)

// **
// Start with some router functions
// **

func handleSlackAlert(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	if settings.Config.SlackURL != "" {
		logging.SlackAlert(settings.Config.SlackURL, params["message"])
	} else {
		json.NewEncoder(w).Encode("Slack URL not set in config.")
	}

	// Echo back message
	json.NewEncoder(w).Encode(params["message"])
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

	//
	// Ping routes
	//
	router.HandleFunc("/ping/{device}", logging.Ping).Methods("POST")

	//
	// GPS Routes
	//
	router.HandleFunc("/session/gps", gps.HandleGet).Methods("GET")
	router.HandleFunc("/session/gps", gps.HandleSetGPS).Methods("POST")
	router.HandleFunc("/session/timezone", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(formatting.JSONResponse{Output: gps.GetTimezone(), Status: "success", OK: true})
	}).Methods("GET")

	//
	// Session routes
	//
	router.HandleFunc("/session", sessions.HandleGetSession).Methods("GET")
	router.HandleFunc("/session/socket", sessions.GetSessionSocket).Methods("GET")
	router.HandleFunc("/session/{name}", sessions.HandleGetSessionValue).Methods("GET")
	router.HandleFunc("/session/{name}", sessions.HandlePostSessionValue).Methods("POST")

	//
	// Settings routes
	//
	router.HandleFunc("/settings", settings.HandleGetAll).Methods("GET")
	router.HandleFunc("/settings/{component}", settings.HandleGet).Methods("GET")
	router.HandleFunc("/settings/{component}/{name}", settings.HandleGetValue).Methods("GET")
	router.HandleFunc("/settings/{component}/{name}/{value}", settings.HandleSet).Methods("POST")

	//
	// PyBus Routes
	//
	router.HandleFunc("/pybus/{src}/{dest}/{data}", pybus.StartRoutine).Methods("POST")
	router.HandleFunc("/pybus/{command}", pybus.StartRoutine).Methods("GET")

	//
	// Serial routes
	//
	router.HandleFunc("/serial/{command}", WriteSerialHandler).Methods("POST")

	//
	// Bluetooth routes
	//
	router.HandleFunc("/bluetooth", bluetooth.GetDeviceInfo).Methods("GET")
	router.HandleFunc("/bluetooth/getDeviceInfo", bluetooth.GetDeviceInfo).Methods("GET")
	router.HandleFunc("/bluetooth/getMediaInfo", bluetooth.GetMediaInfo).Methods("GET")
	router.HandleFunc("/bluetooth/connect", bluetooth.Connect).Methods("GET")
	router.HandleFunc("/bluetooth/disconnect", bluetooth.Disconnect).Methods("GET")
	router.HandleFunc("/bluetooth/prev", bluetooth.Prev).Methods("GET")
	router.HandleFunc("/bluetooth/next", bluetooth.Next).Methods("GET")
	router.HandleFunc("/bluetooth/pause", bluetooth.Pause).Methods("GET")
	router.HandleFunc("/bluetooth/play", bluetooth.Play).Methods("GET")
	router.HandleFunc("/bluetooth/refresh", bluetooth.ForceRefresh).Methods("GET")

	//
	// Status Routes
	//
	router.HandleFunc("/status", logging.GetStatus).Methods("GET")
	router.HandleFunc("/status/{name}", logging.GetStatusValue).Methods("GET")
	router.HandleFunc("/status/{name}", logging.SetStatus).Methods("POST")
	router.HandleFunc("/alert/{message}", handleSlackAlert).Methods("GET")

	//
	// Catch-Alls for (hopefully) a pre-approved pybus function
	// i.e. /doors/lock
	//
	router.HandleFunc("/{device}/{command}", sessions.ParseCommand).Methods("GET")

	//
	// Finally, welcome and meta routes
	//
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "Welcome to MDroid! This port is fully operational, see the docs for applicable routes.", Status: "success", OK: true})
	}).Methods("GET")

	log.Fatal(http.ListenAndServe(":5353", router))
}

// authMiddleware will match http bearer token again the one hardcoded in our config
/*
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		reqToken := r.Header.Get("Authorization")
		splitToken := strings.Split(reqToken, "Bearer")
		if len(splitToken) != 2 || strings.TrimSpace(splitToken[1]) != settings.Config.AuthToken {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403 - Invalid Auth Token!"))
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}*/
