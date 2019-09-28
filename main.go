package main

import (
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
)

// mainStatus will control logging and reporting of status / warnings / errors
var mainStatus logging.ProgramStatus

func init() {
	mainStatus = logging.NewStatus("Main")
}

// define our router and subsequent routes here
func main() {

	// Run through the config file and set up some global variables
	parseConfig()

	// Define routes and begin routing
	startRouter()
}
