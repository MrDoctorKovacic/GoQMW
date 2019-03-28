package bluetooth

import (
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

var btAddress string

// SetAddress makes address given in args available to all dbus functions
func SetAddress(address string) {
	// Format address for dbus
	if btAddress != "" {
		btAddress = strings.Replace(btAddress, ":", "_", -1)
	} else {

	}
}

// Connect new bluetooth device
func Connect(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("dbus-send", "--system", "--print-reply", "--type=method_call", "--dest=org.bluez", "/org/bluez/hci0/dev_"+btAddress+"/player0", "org.bluez.Device1.Connect").Output()

	if err != nil {
		log.Println(err)
		json.NewEncoder(w).Encode(err)
	} else {
		json.NewEncoder(w).Encode(out)
	}
}

// GetDeviceInfo attempts to get metadata about connected device
func GetDeviceInfo(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("dbus-send", "--system", "--print-reply", "--type=method_call", "--dest=org.bluez", "/org/bluez/hci0/dev_"+btAddress+"/player0", "org.freedesktop.DBus.Properties.Get", "string:org.bluez.MediaPlayer1", "string:Status").Output()

	if err != nil {
		log.Println(err)
		json.NewEncoder(w).Encode(err)
	} else {
		json.NewEncoder(w).Encode(out)
	}
}

// GetMediaInfo attempts to get metadata about current track
func GetMediaInfo(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("dbus-send", "--system", "--print-reply", "--type=method_call", "--dest=org.bluez", "/org/bluez/hci0/dev_"+btAddress+"/player0", "org.freedesktop.DBus.Properties.Get", "string:org.bluez.MediaPlayer1", "string:Track").Output()

	if err != nil {
		log.Println(err)
		json.NewEncoder(w).Encode(err)
	} else {
		json.NewEncoder(w).Encode(out)
	}
}

// Prev skips to previous track
func Prev(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("dbus-send", "--system", "--print-reply", "--type=method_call", "--dest=org.bluez", "/org/bluez/hci0/dev_"+btAddress+"/player0", "org.bluez.MediaPlayer1.Previous").Output()

	if err != nil {
		log.Println(err)
		json.NewEncoder(w).Encode(err)
	} else {
		json.NewEncoder(w).Encode(out)
	}
}

// Next skips to next track
func Next(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("dbus-send", "--system", "--print-reply", "--type=method_call", "--dest=org.bluez", "/org/bluez/hci0/dev_"+btAddress+"/player0", "org.bluez.MediaPlayer1.Next").Output()

	if err != nil {
		log.Println(err)
		json.NewEncoder(w).Encode(err)
	} else {
		json.NewEncoder(w).Encode(out)
	}
}

// Play attempts to play bluetooth media
func Play(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("dbus-send", "--system", "--print-reply", "--type=method_call", "--dest=org.bluez", "/org/bluez/hci0/dev_"+btAddress+"/player0", "org.bluez.MediaPlayer1.Play").Output()

	if err != nil {
		log.Println(err)
		json.NewEncoder(w).Encode(err)
	} else {
		json.NewEncoder(w).Encode(out)
	}
}

// Pause attempts to pause bluetooth media
func Pause(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("dbus-send", "--system", "--print-reply", "--type=method_call", "--dest=org.bluez", "/org/bluez/hci0/dev_"+btAddress+"/player0", "org.bluez.MediaPlayer1.Pause").Output()

	if err != nil {
		log.Println(err)
		json.NewEncoder(w).Encode(err)
	} else {
		json.NewEncoder(w).Encode(out)
	}
}
