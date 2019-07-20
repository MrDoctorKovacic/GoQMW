package main

import (
	"github.com/MrDoctorKovacic/MDroid-Core/status"
)

// MainStatus will control logging and reporting of status / warnings / errors
var MainStatus = status.NewStatus("Main")

// define our router and subsequent routes here
func main() {

	// Run through the config file and set up some global variables
	config := parseConfig()

	// Define routes and begin routing
	startRouter(config)
}
