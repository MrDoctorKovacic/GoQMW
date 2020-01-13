package main

func main() {
	Start()
}

// Start is exported for later modular usage
func Start() {
	// Run through the config file and set up some global variables
	router := parseConfig()

	// Define routes and begin routing
	startRouter(router)
}
