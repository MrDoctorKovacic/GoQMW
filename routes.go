package main

import (
	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/routes/serial"
	"github.com/qcasey/MDroid-Core/routes/shutdown"
	"github.com/rs/zerolog/log"
)

// addRoutes initializes an MDroid router with default system routes
func addRoutes(router *mux.Router) {
	log.Info().Msg("Configuring module routes...")

	//
	// Module Routes
	//
	router.HandleFunc("/shutdown", shutdown.HandleShutdown).Methods("GET")
	router.HandleFunc("/serial/{command}", serial.WriteSerialHandler).Methods("POST", "GET")
}
