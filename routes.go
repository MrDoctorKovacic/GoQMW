package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/MrDoctorKovacic/MDroid-Core/bluetooth"
	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/gps"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/pybus"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/gorilla/mux"
)

// **
// Start with some router functions
// **

func handleSlackAlert(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	if Config.SlackURL != "" {
		logging.SlackAlert(Config.SlackURL, params["message"])
	} else {
		json.NewEncoder(w).Encode("Slack URL not set in config.")
	}

	// Echo back message
	json.NewEncoder(w).Encode(params["message"])
}

// welcomeRoute intros MDroid-Core, proving port and service works
func welcomeRoute(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "Welcome to MDroid! This port is fully operational, see the docs for applicable routes.", Status: "success", OK: true})
}

// handleSetGPS posts a new GPS fix
func handleSetGPS(w http.ResponseWriter, r *http.Request) {
	var newdata gps.Fix
	_ = json.NewDecoder(r.Body).Decode(&newdata)
	postingString := Config.Location.Set(newdata)

	// Insert into database
	if postingString != "" && Config.DatabaseEnabled {
		online, err := Config.DB.Write(fmt.Sprintf("gps %s", strings.TrimSuffix(postingString, ",")))

		if err != nil && online {
			mainStatus.Log(logging.Error(), fmt.Sprintf("Error writing string %s to influx DB: %s", postingString, err.Error()))
		} else if Config.VerboseOutput {
			mainStatus.Log(logging.OK(), fmt.Sprintf("Logged %s to database", postingString))
		}
	}
}

// **
// end router functions
// **

// Configures routes, starts router with optional middleware if configured
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
	// Session routes
	//
	router.HandleFunc("/session", MainSession.HandleGetSession).Methods("GET")
	router.HandleFunc("/session/socket", MainSession.GetSessionSocket).Methods("GET")
	router.HandleFunc("/session/gps", Config.Location.HandleGet).Methods("GET")
	router.HandleFunc("/session/gps", handleSetGPS).Methods("POST")
	router.HandleFunc("/session/{name}", MainSession.HandleGetSessionValue).Methods("GET")
	router.HandleFunc("/session/{name}", MainSession.HandlePostSessionValue).Methods("POST")

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
	router.HandleFunc("/{device}/{command}", MainSession.ParseCommand).Methods("GET")

	//
	// Finally, welcome and meta routes
	//
	router.HandleFunc("/", welcomeRoute).Methods("GET")

	if Config.DebugSessionFile != "" {
		// Log all routes for debugging later, if enabled
		// The locks here slow things down, should only be used to generate a run file, not in production
		enabled, err := logging.EnableLogging(Config.DebugSessionFile, Config.Timezone)
		if enabled {
			router.Use(logging.LogMiddleware)
		} else {
			mainStatus.Log(logging.Error(), "Failed to open debug file, is it writable?")
			mainStatus.Log(logging.Error(), err.Error())
		}
	}

	if Config.AuthToken != "" {
		// Ask for matching Auth Token before taking requests
		router.Use(authMiddleware)
	}

	log.Fatal(http.ListenAndServe(":5353", router))
}

// authMiddleware will match http bearer token again the one hardcoded in our config
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		reqToken := r.Header.Get("Authorization")
		splitToken := strings.Split(reqToken, "Bearer")
		if len(splitToken) != 2 || strings.TrimSpace(splitToken[1]) != Config.AuthToken {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403 - Invalid Auth Token!"))
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}
