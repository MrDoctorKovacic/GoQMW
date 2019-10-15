package main

import (
	"fmt"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
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

// Some shutdowns are more complicated than others, ensure we shut down safely
func gracefulShutdown(name string) {
	serialCommand := fmt.Sprintf("powerOff%s", name)

	switch name {
	case "Board":
		mserial.CommandNetworkMachine("etc", "shutdown")
		mserial.MachineShutdown(settings.Config.SerialControlDevice, "board", time.Second*10, serialCommand)
	case "Wireless":
		mserial.MachineShutdown(settings.Config.SerialControlDevice, "lte", time.Second*10, serialCommand)
	default:
		mserial.Push(settings.Config.SerialControlDevice, serialCommand)
	}
}
