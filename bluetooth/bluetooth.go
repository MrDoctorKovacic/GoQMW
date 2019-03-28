package bluetooth

import (
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"strings"

	"github.com/godbus/dbus"
)

var btAddress string

func init() {
	flag.StringVar(&btAddress, "bt-device", "", "Bluetooth Media device to accept as default")
	flag.Parse()

	// Format address for dbus
	if btAddress != "" {
		btAddress = strings.Replace(btAddress, ":", "_", -1)
	} else {

	}
}

// fetches a dbus object
func _getObject() (dbus.BusObject, error) {
	if btAddress != "" {
		conn, err := dbus.SessionBus()

		if err != nil {
			return nil, err
		}

		obj := conn.Object("org.bluez", dbus.ObjectPath("/org/bluez/hci0/dev_4C_32_75_AD_98_24/player0"))
		return obj, nil
	}

	return nil, errors.New("Invalid BT Address: " + btAddress)
}

// Connect new bluetooth device
func Connect(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("test")
}

// GetDeviceInfo attempts to get metadata about connected device
func GetDeviceInfo(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("test")
}

// GetMediaInfo attempts to get metadata about current track
func GetMediaInfo(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("test")
}

// Prev skips to previous track
func Prev(w http.ResponseWriter, r *http.Request) {
	obj, err := _getObject()

	if err != nil {
		json.NewEncoder(w).Encode(err)
	} else {
		var s string
		err = obj.Call("org.bluez.MediaPlayer1.Previous", 0).Store(&s)

		if err != nil {
			json.NewEncoder(w).Encode(err)
		} else {
			json.NewEncoder(w).Encode(s)
		}
	}
}

// Next skips to next track
func Next(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("test")
}

// Play attempts to play bluetooth media
func Play(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("test")
}

// Pause attempts to pause bluetooth media
func Pause(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("test")
}
