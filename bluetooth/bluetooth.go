// Package bluetooth is a rudimentary interface between MDroid-Core and underlying BT dbus
package bluetooth

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/format"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/gosimple/slug"
	"github.com/rs/zerolog/log"
)

// Regex expressions for parsing dbus output
var (
	BluetoothAddress string
	tmuxStarted      bool
	replySerialRegex *regexp.Regexp
	findStringRegex  *regexp.Regexp
	cleanRegex       *regexp.Regexp
)

func init() {
	replySerialRegex = regexp.MustCompile(`(.*reply_serial=2\n\s*variant\s*)array`)
	findStringRegex = regexp.MustCompile(`string\s"(.*)"|uint32\s(\d)+`)
	cleanRegex = regexp.MustCompile(`(string|uint32|\")+`)
}

// Setup bluetooth with address
func Setup(configAddr *map[string]string) {
	configMap := *configAddr
	bluetoothAddress, usingBluetooth := configMap["BLUETOOTH_ADDRESS"]
	if !usingBluetooth {
		log.Warn().Msg("No bluetooth address found in config, using empty address")
		BluetoothAddress = ""
		return
	}

	SetAddress(bluetoothAddress)
	log.Info().Msg("Enabling auto refresh of BT address")
	go startAutoRefresh()
}

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
		BluetoothAddress = strings.Replace(strings.TrimSpace(address), ":", "_", -1)
		log.Info().Msg("Now routing Bluetooth commands to " + BluetoothAddress)

		// Set new address to persist in settings file
		settings.Set("CONFIG", "BLUETOOTH_ADDRESS", BluetoothAddress)
	}
}

// Connect bluetooth device
func Connect(w http.ResponseWriter, r *http.Request) {
	ScanOn()
	log.Info().Msg("Connecting to bluetooth device...")
	time.Sleep(13 * time.Second)

	SendDBusCommand(
		[]string{"/org/bluez/hci0/dev_" + BluetoothAddress, "org.bluez.Device1.Connect"},
		false,
		true)

	log.Info().Msg("Connection successful.")
	format.WriteResponse(&w, r, format.JSONResponse{Output: "OK", OK: true})
}

// HandleDisconnect bluetooth device
func HandleDisconnect(w http.ResponseWriter, r *http.Request) {
	err := Disconnect()
	if err != nil {
		log.Error().Msg(err.Error())
		format.WriteResponse(&w, r, format.JSONResponse{Output: "Could not lookup user", OK: false})
	}
	format.WriteResponse(&w, r, format.JSONResponse{Output: "OK", OK: true})
}

// Disconnect bluetooth device
func Disconnect() error {
	log.Info().Msg("Disconnecting from bluetooth device...")

	SendDBusCommand(
		[]string{"/org/bluez/hci0/dev_" + BluetoothAddress,
			"org.bluez.Device1.Disconnect"},
		false,
		true)

	return nil
}

func askDeviceInfo() map[string]string {
	log.Info().Msg("Getting device info...")

	deviceMessage := []string{"/org/bluez/hci0/dev_" + BluetoothAddress + "/player0", "org.freedesktop.DBus.Properties.Get", "string:org.bluez.MediaPlayer1", "string:Status"}
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
	mediaMessage := []string{"/org/bluez/hci0/dev_" + BluetoothAddress + "/player0", "org.freedesktop.DBus.Properties.Get", "string:org.bluez.MediaPlayer1", "string:Track"}
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
		format.WriteResponse(&w, r, format.JSONResponse{Output: "Error getting media info", Status: "fail", OK: false})
		return
	}
	format.WriteResponse(&w, r, format.JSONResponse{Output: deviceStatus, Status: "success", OK: true})
}

// GetMediaInfo attempts to get metadata about current track
func GetMediaInfo(w http.ResponseWriter, r *http.Request) {
	deviceStatus := askDeviceInfo()
	if deviceStatus == nil {
		format.WriteResponse(&w, r, format.JSONResponse{Output: "Error getting media info", Status: "fail", OK: false})
		return
	}

	response := askMediaInfo()
	if response == nil {
		format.WriteResponse(&w, r, format.JSONResponse{Output: "Error getting media info", Status: "fail", OK: false})
		return
	}

	// Append device status to media info
	response["Status"] = deviceStatus["Meta"]

	// Append Album / Artwork slug if both exist
	album, albumOK := response["Album"]
	artist, artistOK := response["Artist"]
	if albumOK && artistOK {
		response["Album_Artwork"] = slug.Make(artist) + "/" + slug.Make(album) + ".jpg"
	}

	// Echo back all info
	format.WriteResponse(&w, r, format.JSONResponse{Output: response, Status: "success", OK: true})
}

// Prev skips to previous track
func Prev(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Going to previous track...")
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + BluetoothAddress + "/player0", "org.bluez.MediaPlayer1.Previous"}, false, false)
	format.WriteResponse(&w, r, format.JSONResponse{Output: "OK", OK: true})
}

// Next skips to next track
func Next(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Going to next track...")
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + BluetoothAddress + "/player0", "org.bluez.MediaPlayer1.Next"}, false, false)
	format.WriteResponse(&w, r, format.JSONResponse{Output: "OK", OK: true})
}

// Play attempts to play bluetooth media
func Play(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Attempting to play media...")
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + BluetoothAddress + "/player0", "org.bluez.MediaPlayer1.Play"}, false, false)
	format.WriteResponse(&w, r, format.JSONResponse{Output: "OK", OK: true})
}

// Pause attempts to pause bluetooth media
func Pause(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Attempting to pause media...")
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + BluetoothAddress + "/player0", "org.bluez.MediaPlayer1.Pause"}, false, false)
	format.WriteResponse(&w, r, format.JSONResponse{Output: "OK", OK: true})
}
