package gyro

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/format/response"
)

// Module exports MDroid module
type Module struct{}

// Measurement contains a simple X,Y,Z output from the IMU
type Measurement struct {
	X float64 `json:"X"`
	Y float64 `json:"Y"`
	Z float64 `json:"Z"`
}

type gyros struct {
	Acceleration Measurement `json:"Acceleration,omitempty"`
	Gyroscope    Measurement `json:"Gyroscope,omitempty"`
	Magnetic     Measurement `json:"Magnetic,omitempty"`
}

var (
	// Mod exports our module functionality
	Mod                Module
	currentGyroReading gyros
)

// Setup handles module init
func (*Module) Setup(configAddr *map[string]string) {
}

// SetRoutes inits module routes
func (*Module) SetRoutes(router *mux.Router) {
	//
	// Serial routes
	//
	router.HandleFunc("/session/gyros", getGyroMeasurements).Methods("GET")
}

// AddMeasurement to current readings
func AddMeasurement(name string, m Measurement) error {
	switch name {
	case "ACCELERATION":
		currentGyroReading.Acceleration = m
	case "GYROSCOPE":
		currentGyroReading.Gyroscope = m
	case "MAGNETIC":
		currentGyroReading.Magnetic = m
	default:
		return fmt.Errorf("Measurement name %s not registered for input", name)
	}
	return nil
}

// getGyroMeasurements handles messages sent through the server
func getGyroMeasurements(w http.ResponseWriter, r *http.Request) {
	response.WriteNew(&w, r, response.JSONResponse{Output: currentGyroReading, OK: true})
}
