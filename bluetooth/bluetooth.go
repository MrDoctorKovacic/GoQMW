// Package bluetooth is a rudimentary interface between MDroid-Core and underlying BT dbus
package bluetooth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/format/response"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/gosimple/slug"
	"github.com/rs/zerolog/log"
)

// startAutoRefresh will begin go routine for refreshing bt media device address
func startAutoRefresh() {
	for {
		getConnectedAddress()
		time.Sleep(1000 * time.Millisecond)
	}
}

// ForceRefresh to immediately reload bt address
func ForceRefresh(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Forcing refresh of BT address")
	go getConnectedAddress()
}

// SetAddress makes address given in args available to all dbus functions
func SetAddress(address string) {
	// Format address for dbus
	if address != "" {
		Mod.BluetoothAddress = strings.Replace(strings.TrimSpace(address), ":", "_", -1)
		log.Info().Msg("Now routing Bluetooth commands to " + Mod.BluetoothAddress)

		// Set new address to persist in settings file
		settings.Set("CONFIG", "BLUETOOTH_ADDRESS", Mod.BluetoothAddress)
	}
}

// HandleConnect wrapper for connect
func HandleConnect(w http.ResponseWriter, r *http.Request) {
	Connect()
	response.WriteNew(&w, r, response.JSONResponse{Output: "OK", OK: true})
}

// Connect bluetooth device
func Connect() {
	ScanOn()
	log.Info().Msg("Connecting to bluetooth device...")
	time.Sleep(5 * time.Second)

	SendDBusCommand(
		[]string{"/org/bluez/hci0/dev_" + Mod.BluetoothAddress, "org.bluez.Device1.Connect"},
		false,
		true)

	log.Info().Msg("Connection successful.")
}

// HandleDisconnect bluetooth device
func HandleDisconnect(w http.ResponseWriter, r *http.Request) {
	err := Disconnect()
	if err != nil {
		log.Error().Msg(err.Error())
		response.WriteNew(&w, r, response.JSONResponse{Output: "Could not lookup user", OK: false})
	}
	response.WriteNew(&w, r, response.JSONResponse{Output: "OK", OK: true})
}

// Disconnect bluetooth device
func Disconnect() error {
	log.Info().Msg("Disconnecting from bluetooth device...")

	SendDBusCommand(
		[]string{"/org/bluez/hci0/dev_" + Mod.BluetoothAddress,
			"org.bluez.Device1.Disconnect"},
		false,
		true)

	return nil
}

func askDeviceInfo() map[string]string {
	log.Info().Msg("Getting device info...")

	deviceMessage := []string{"/org/bluez/hci0/dev_" + Mod.BluetoothAddress + "/player0", "org.freedesktop.DBus.Properties.Get", "string:org.bluez.MediaPlayer1", "string:Status"}
	result, ok := SendDBusCommand(deviceMessage, true, false)
	if !ok {
		return nil
	}
	if result == "" {
		// empty response
		log.Warn().Msg(fmt.Sprintf("Empty dbus response when querying device, not attempting to clean. We asked: \n%s", strings.Join(deviceMessage, " ")))
		return nil
	}
	return cleanDBusOutput(result)
}

func askMediaInfo() map[string]string {
	log.Info().Msg("Getting media info...")
	mediaMessage := []string{"/org/bluez/hci0/dev_" + Mod.BluetoothAddress + "/player0", "org.freedesktop.DBus.Properties.Get", "string:org.bluez.MediaPlayer1", "string:Track"}
	result, ok := SendDBusCommand(mediaMessage, true, false)
	if !ok {
		return nil
	}
	if result == "" {
		// empty response
		log.Warn().Msg(fmt.Sprintf("Empty dbus response when querying media, not attempting to clean. We asked: \n%s", strings.Join(mediaMessage, " ")))
		return nil
	}
	return cleanDBusOutput(result)
}

// GetDeviceInfo attempts to get metadata about connected device
func GetDeviceInfo(w http.ResponseWriter, r *http.Request) {
	deviceStatus := askDeviceInfo()
	if deviceStatus == nil {
		response.WriteNew(&w, r, response.JSONResponse{Output: "Error getting media info", Status: "fail", OK: false})
		return
	}
	response.WriteNew(&w, r, response.JSONResponse{Output: deviceStatus, Status: "success", OK: true})
}

// GetMediaInfo attempts to get metadata about current track
func GetMediaInfo(w http.ResponseWriter, r *http.Request) {
	deviceStatus := askDeviceInfo()
	if deviceStatus == nil {
		response.WriteNew(&w, r, response.JSONResponse{Output: "Error getting media info", Status: "fail", OK: false})
		return
	}

	resp := askMediaInfo()
	if resp == nil {
		response.WriteNew(&w, r, response.JSONResponse{Output: "Error getting media info", Status: "fail", OK: false})
		return
	}

	// Append device status to media info
	resp["Status"] = deviceStatus["Meta"]

	// Append Album / Artwork slug if both exist
	album, albumOK := resp["Album"]
	artist, artistOK := resp["Artist"]
	if albumOK && artistOK {
		resp["Album_Artwork"] = slug.Make(artist) + "/" + slug.Make(album) + ".jpg"
	}

	// Echo back all info
	response.WriteNew(&w, r, response.JSONResponse{Output: resp, Status: "success", OK: true})
}

// Prev skips to previous track
func Prev(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Going to previous track...")
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + Mod.BluetoothAddress + "/player0", "org.bluez.MediaPlayer1.Previous"}, false, false)
	response.WriteNew(&w, r, response.JSONResponse{Output: "OK", OK: true})
}

// Next skips to next track
func Next(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Going to next track...")
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + Mod.BluetoothAddress + "/player0", "org.bluez.MediaPlayer1.Next"}, false, false)
	response.WriteNew(&w, r, response.JSONResponse{Output: "OK", OK: true})
}

// HandlePlay attempts to play bluetooth media
func HandlePlay(w http.ResponseWriter, r *http.Request) {
	Play()
	response.WriteNew(&w, r, response.JSONResponse{Output: "OK", OK: true})
}

// Play attempts to play bluetooth media
func Play() {
	log.Info().Msg("Attempting to play media...")
	SendDBusCommand([]string{"/org/bluez/hci0/dev_" + Mod.BluetoothAddress + "/player0", "org.bluez.MediaPlayer1.Play"}, false, false)
}

// HandlePause attempts to pause bluetooth media
func HandlePause(w http.ResponseWriter, r *http.Request) {
	Pause()
	response.WriteNew(&w, r, response.JSONResponse{Output: "OK", OK: true})
}

// Pause attempts to pause bluetooth media
func Pause() {
	log.Info().Msg("Attempting to pause media...")
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + Mod.BluetoothAddress + "/player0", "org.bluez.MediaPlayer1.Pause"}, false, false)
}
