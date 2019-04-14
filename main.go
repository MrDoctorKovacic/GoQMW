package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os/exec"

	"github.com/MrDoctorKovacic/GoQMW/external/bluetooth"
	"github.com/MrDoctorKovacic/GoQMW/external/ping"
	"github.com/MrDoctorKovacic/GoQMW/external/pybus"
	"github.com/MrDoctorKovacic/GoQMW/influx"
	"github.com/MrDoctorKovacic/GoQMW/sessions"
	"github.com/gorilla/mux"
)

// Reboot the machine
func reboot(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("OK")
	exec.Command("reboot", "now")
}

// define our router and subsequent routes here
func main() {

	// Start with program arguments
	var (
		sessionFile       string
		dbHost            string
		dbName            string
		btAddress         string
		remotePingAddress string
	)
	flag.StringVar(&sessionFile, "session-file", "", "File to save and recover the last-known session to.")
	flag.StringVar(&dbHost, "db-host", "", "Influx Database fully qualified url on localhost to log with")
	flag.StringVar(&dbName, "db-name", "", "Influx Database name on localhost to log with")
	flag.StringVar(&btAddress, "bt-device", "", "Bluetooth Media device to connect and use as default")
	flag.StringVar(&remotePingAddress, "ping-host", "", "Remote address to fwd pings to")
	flag.Parse()

	if dbHost != "" {
		var sqlEnabled = true
		//var tries = 0
		DB := influx.Influx{Host: dbHost, Database: dbName}

		// Initial connection
		/*
			err := DB.Ping()
			// Retry connection every second
			if err != nil {
				for tries < 15 {
					err := DB.Ping()
					if err == nil {
						break
					}
					log.Println("Failed to connect to DB, retrying.")
					time.Sleep(2 * time.Second)
					tries++
				}
				if err != nil {
					log.Println(err.Error())
					sqlEnabled = false
				}
			}*/

		if sqlEnabled {
			// Pass DB pool to imports
			sessions.Setup(DB, sessionFile)

			if remotePingAddress != "" {
				ping.Setup(DB, remotePingAddress)
			} else {
				log.Println("[Ping] Not accepting pings.")
			}
		}
	} else {
		log.Println("[Influx] Not logging to influx db.")
	}

	// Pass argument to its rightful owner
	bluetooth.SetAddress(btAddress)

	// Init router
	router := mux.NewRouter()

	//
	// Main routes
	//
	router.HandleFunc("/restart", reboot).Methods("GET")

	//
	// Ping routes
	//
	router.HandleFunc("/ping/{device}", ping.Ping).Methods("POST")

	//
	// Session routes
	//
	router.HandleFunc("/session", sessions.GetSession).Methods("GET")
	router.HandleFunc("/session/{name}", sessions.GetSessionValue).Methods("GET")
	router.HandleFunc("/session/gps", sessions.SetGPSValue).Methods("POST")
	router.HandleFunc("/session/{name}", sessions.SetSessionValue).Methods("POST")

	//
	// PyBus Routes
	//
	router.HandleFunc("/pybus/{command}", pybus.StartPyBusRoutine).Methods("GET")
	router.HandleFunc("/pybus/restart", pybus.RestartService).Methods("GET")

	//
	// ALPR Routes
	//
	router.HandleFunc("/alpr/{plate}", sessions.LogALPR).Methods("POST")
	router.HandleFunc("/alpr/restart", sessions.RestartALPR).Methods("GET")

	//
	// Bluetooth routes
	//
	router.HandleFunc("/bluetooth", bluetooth.GetDeviceInfo).Methods("GET")
	router.HandleFunc("/bluetooth/getDeviceInfo", bluetooth.GetDeviceInfo).Methods("GET")
	router.HandleFunc("/bluetooth/getMediaInfo", bluetooth.GetMediaInfo).Methods("GET")
	router.HandleFunc("/bluetooth/connect", bluetooth.Connect).Methods("GET")
	router.HandleFunc("/bluetooth/prev", bluetooth.Prev).Methods("GET")
	router.HandleFunc("/bluetooth/next", bluetooth.Next).Methods("GET")
	router.HandleFunc("/bluetooth/pause", bluetooth.Pause).Methods("GET")
	router.HandleFunc("/bluetooth/play", bluetooth.Play).Methods("GET")
	router.HandleFunc("/bluetooth/restart", bluetooth.RestartService).Methods("GET")

	log.Fatal(http.ListenAndServe(":5353", router))
}
