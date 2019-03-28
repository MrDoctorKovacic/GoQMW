package main

import (
	"flag"
	"log"
	"net/http"

	b "github.com/MrDoctorKovacic/GoQMW/bluetooth"
	s "github.com/MrDoctorKovacic/GoQMW/sessions"
	"github.com/gorilla/mux"
)

func init() {

	// Start with program arguments
	var (
		sqlHost     string
		sqlDatabase string
		sqlUser     string
		sqlPassword string
		btAddress   string
	)
	flag.StringVar(&sqlHost, "host", "localhost", "SQL Host, defaults to localhost")
	flag.StringVar(&sqlDatabase, "database", "", "SQL Database, defaults to ")
	flag.StringVar(&sqlUser, "user", "", "SQL Username")
	flag.StringVar(&sqlPassword, "password", "", "SQL Password")
	flag.StringVar(&btAddress, "bt-device", "", "Bluetooth Media device to connect and use as default")
	flag.Parse()

	// Pass arguments to their rightful owners
	s.SQLConnect(sqlHost, sqlDatabase, sqlUser, sqlPassword)
	b.SetAddress(btAddress)
}

// define our router and subsequent routes here
func main() {
	// Init router
	router := mux.NewRouter()

	// Session routes
	router.HandleFunc("/session", s.GetSession).Methods("GET")
	router.HandleFunc("/session/{Name}", s.GetSessionValue).Methods("GET")
	router.HandleFunc("/session/{Name}", s.UpdateSessionValue).Methods("POST")

	// PyBus Route
	router.HandleFunc("/pybus/{command}", s.PyBus).Methods("GET")

	// Bluetooth routes
	router.HandleFunc("/bluetooth", b.GetDeviceInfo).Methods("GET")
	router.HandleFunc("/bluetooth/getDeviceInfo", b.GetDeviceInfo).Methods("GET")
	router.HandleFunc("/bluetooth/getMediaInfo", b.GetMediaInfo).Methods("GET")
	router.HandleFunc("/bluetooth/connect", b.Connect).Methods("GET")
	router.HandleFunc("/bluetooth/prev", b.Prev).Methods("GET")
	router.HandleFunc("/bluetooth/next", b.Next).Methods("GET")
	router.HandleFunc("/bluetooth/pause", b.Pause).Methods("GET")
	router.HandleFunc("/bluetooth/play", b.Play).Methods("GET")

	log.Fatal(http.ListenAndServe(":5353", router))
}
