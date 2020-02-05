package main

import (
	"github.com/gorilla/mux"
	bluetooth "github.com/qcasey/MDroid-Bluetooth"
	"github.com/qcasey/MDroid-Core/db"
	"github.com/qcasey/MDroid-Core/mserial"
	"github.com/qcasey/MDroid-Core/pybus"
	"github.com/qcasey/MDroid-Core/sessions/gps"
	"github.com/qcasey/MDroid-Core/sessions/system"
)

func main() {
	// Init router
	router := mux.NewRouter()

	SetDefaultRoutes(router)

	// Run through the config file and retrieve some settings
	configMap := parseConfig()

	// Setup conventional modules
	// TODO: More modular handling of modules
	mserial.Mod.Setup(configMap)
	mserial.Mod.SetRoutes(router)
	bluetooth.Mod.Setup(configMap)
	bluetooth.Mod.SetRoutes(router)
	gps.Mod.Setup(configMap)
	gps.Mod.SetRoutes(router)
	pybus.Mod.Setup(configMap)
	pybus.Mod.SetRoutes(router)
	system.Mod.Setup(configMap)
	system.Mod.SetRoutes(router)
	db.Mod.Setup(configMap)

	// Connect bluetooth device on startup
	bluetooth.Connect()

	Start(router)
}
