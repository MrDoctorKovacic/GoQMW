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
	}
	log.Println("[Bluetooth] Accepting connections initially from " + btAddress)
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

// Connect new bluetooth device
func Connect(w http.ResponseWriter, r *http.Request) {
	if btAddress != "" {
		out, err := exec.Command("dbus-send", "--system", "--print-reply", "--type=method_call", "--dest=org.bluez", "/org/bluez/hci0/dev_"+btAddress+"/player0", "org.bluez.Device1.Connect").Output()

		if err != nil {
			log.Println(err)
			json.NewEncoder(w).Encode(err)
		} else {
			json.NewEncoder(w).Encode(out)
		}
	} else {
		json.NewEncoder(w).Encode("No valid BT Address to run command")
	}
}

// GetDeviceInfo attempts to get metadata about connected device
func GetDeviceInfo(w http.ResponseWriter, r *http.Request) {
	if btAddress != "" {
		out, err := exec.Command("dbus-send", "--system", "--print-reply", "--type=method_call", "--dest=org.bluez", "/org/bluez/hci0/dev_"+btAddress+"/player0", "org.freedesktop.DBus.Properties.Get", "string:org.bluez.MediaPlayer1", "string:Status").Output()

		if err != nil {
			log.Println(err)
			json.NewEncoder(w).Encode(err)
		} else {
			json.NewEncoder(w).Encode(out)
		}
	} else {
		json.NewEncoder(w).Encode("No valid BT Address to run command")
	}
}

// GetMediaInfo attempts to get metadata about current track
func GetMediaInfo(w http.ResponseWriter, r *http.Request) {
	if btAddress != "" {
		out, err := exec.Command("dbus-send", "--system", "--print-reply", "--type=method_call", "--dest=org.bluez", "/org/bluez/hci0/dev_"+btAddress+"/player0", "org.freedesktop.DBus.Properties.Get", "string:org.bluez.MediaPlayer1", "string:Track").Output()

		if err != nil {
			log.Println(err)
			json.NewEncoder(w).Encode(err)
		} else {
			json.NewEncoder(w).Encode(out)
		}
	} else {
		json.NewEncoder(w).Encode("No valid BT Address to run command")
	}
}

// Prev skips to previous track
func Prev(w http.ResponseWriter, r *http.Request) {
	if btAddress != "" {
		out, err := exec.Command("dbus-send", "--system", "--print-reply", "--type=method_call", "--dest=org.bluez", "/org/bluez/hci0/dev_"+btAddress+"/player0", "org.bluez.MediaPlayer1.Previous").Output()

		if err != nil {
			log.Println(err)
			json.NewEncoder(w).Encode(err)
		} else {
			json.NewEncoder(w).Encode(out)
		}
	} else {
		json.NewEncoder(w).Encode("No valid BT Address to run command")
	}
}

// Next skips to next track
func Next(w http.ResponseWriter, r *http.Request) {
	if btAddress != "" {
		out, err := exec.Command("dbus-send", "--system", "--print-reply", "--type=method_call", "--dest=org.bluez", "/org/bluez/hci0/dev_"+btAddress+"/player0", "org.bluez.MediaPlayer1.Next").Output()

		if err != nil {
			log.Println(err)
			json.NewEncoder(w).Encode(err)
		} else {
			json.NewEncoder(w).Encode(out)
		}
	} else {
		json.NewEncoder(w).Encode("No valid BT Address to run command")
	}
}

// Play attempts to play bluetooth media
func Play(w http.ResponseWriter, r *http.Request) {
	if btAddress != "" {
		out, err := exec.Command("dbus-send", "--system", "--print-reply", "--type=method_call", "--dest=org.bluez", "/org/bluez/hci0/dev_"+btAddress+"/player0", "org.bluez.MediaPlayer1.Play").Output()

		if err != nil {
			log.Println(err)
			json.NewEncoder(w).Encode(err)
		} else {
			json.NewEncoder(w).Encode(out)
		}
	} else {
		json.NewEncoder(w).Encode("No valid BT Address to run command")
	}
}

// Pause attempts to pause bluetooth media
func Pause(w http.ResponseWriter, r *http.Request) {
	if btAddress != "" {
		out, err := exec.Command("dbus-send", "--system", "--print-reply", "--type=method_call", "--dest=org.bluez", "/org/bluez/hci0/dev_"+btAddress+"/player0", "org.bluez.MediaPlayer1.Pause").Output()

		if err != nil {
			log.Println(err)
			json.NewEncoder(w).Encode(err)
		} else {
			json.NewEncoder(w).Encode(out)
		}
	} else {
		json.NewEncoder(w).Encode("No valid BT Address to run command")
	}
}
