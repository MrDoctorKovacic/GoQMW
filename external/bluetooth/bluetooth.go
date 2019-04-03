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
	if address != "" {
		btAddress = strings.Replace(address, ":", "_", -1)
		log.Println("[Bluetooth] Accepting connections initially from " + btAddress)
	}
}

// RestartService will attempt to restart the bluetooth service
func RestartService(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("/home/pi/le/auto/restart_bluetooth.sh").Output()

	if err != nil {
		log.Println(err)
		json.NewEncoder(w).Encode(err)
	} else {
		json.NewEncoder(w).Encode(out)
	}
}

// SendDBusCommand used as a general BT control function for these endpoints
func SendDBusCommand(args []string) string {
	if btAddress != "" {
		// Fill in the meta nonsense
		args = append([]string{"--system", "--print-reply", "--type=method_call", "--dest=org.bluez"}, args...)
		out, err := exec.Command("dbus-send", args...).Output()

		if err != nil {
			log.Println(err)
			return err.Error()
		}

		return string(out)
	}

	return "No valid BT Address to run command"
}

// Connect new bluetooth device
func Connect(w http.ResponseWriter, r *http.Request) {
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + btAddress, "org.bluez.Device1.Connect"})
	json.NewEncoder(w).Encode("OK")
}

// GetDeviceInfo attempts to get metadata about connected device
func GetDeviceInfo(w http.ResponseWriter, r *http.Request) {
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + btAddress + "/player0", "org.freedesktop.DBus.Properties.Get", "string:org.bluez.MediaPlayer1", "string:Status"})
	json.NewEncoder(w).Encode("OK")
}

// GetMediaInfo attempts to get metadata about current track
func GetMediaInfo(w http.ResponseWriter, r *http.Request) {
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + btAddress + "/player0", "org.freedesktop.DBus.Properties.Get", "string:org.bluez.MediaPlayer1", "string:Track"})
	json.NewEncoder(w).Encode("OK")
}

// Prev skips to previous track
func Prev(w http.ResponseWriter, r *http.Request) {
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + btAddress + "/player0", "org.bluez.MediaPlayer1.Previous"})
	json.NewEncoder(w).Encode("OK")
}

// Next skips to next track
func Next(w http.ResponseWriter, r *http.Request) {
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + btAddress + "/player0", "org.bluez.MediaPlayer1.Next"})
	json.NewEncoder(w).Encode("OK")
}

// Play attempts to play bluetooth media
func Play(w http.ResponseWriter, r *http.Request) {
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + btAddress + "/player0", "org.bluez.MediaPlayer1.Play"})
	json.NewEncoder(w).Encode("OK")
}

// Pause attempts to pause bluetooth media
func Pause(w http.ResponseWriter, r *http.Request) {
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + btAddress + "/player0", "org.bluez.MediaPlayer1.Pause"})
	json.NewEncoder(w).Encode("OK")
}
