package mserial

import (
	"fmt"
	"net/http"

	"github.com/qcasey/MDroid-Core/format/response"
	"github.com/qcasey/MDroid-Core/mqtt"
)

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

var currentGyroReading gyros

// addMeasurement to current readings
func addMeasurement(name string, m Measurement) error {
	switch name {
	case "ACCELERATION":
		currentGyroReading.Acceleration = m
		go mqtt.Publish("measurements/accel", fmt.Sprintf("%f,%f,%f", m.X, m.Y, m.Z), false)
	case "GYROSCOPE":
		currentGyroReading.Gyroscope = m
		go mqtt.Publish("measurements/gyro", fmt.Sprintf("%f,%f,%f", m.X, m.Y, m.Z), false)
	case "MAGNETIC":
		currentGyroReading.Magnetic = m
		go mqtt.Publish("measurements/magnetic", fmt.Sprintf("%f,%f,%f", m.X, m.Y, m.Z), false)
	default:
		return fmt.Errorf("Measurement name %s not registered for input", name)
	}
	return nil
}

// getGyroMeasurements handles messages sent through the server
func getGyroMeasurements(w http.ResponseWriter, r *http.Request) {
	response.WriteNew(&w, r, response.JSONResponse{Output: currentGyroReading, OK: true})
}
