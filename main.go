package main

func main() {
	// Run through the config file and set up some global variables
	router := parseConfig()

	// Define routes and begin routing
	startRouter(router)
}
