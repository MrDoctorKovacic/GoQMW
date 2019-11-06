package main

func main() {
	// Run through the config file and set up some global variables
	parseConfig()

	// Define routes and begin routing
	startRouter()
}
