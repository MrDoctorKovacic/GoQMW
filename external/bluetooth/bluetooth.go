package bluetooth

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/external/status"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/gosimple/slug"
)

var btAddress string

// BluetoothStatus will control logging and reporting of status / warnings / errors
var BluetoothStatus = status.NewStatus("Bluetooth")

// Regex expressions for parsing dbus output
var re = regexp.MustCompile(`(.*reply_serial=2\n\s*variant\s*)array`)
var reFind = regexp.MustCompile(`string\s"(.*)"|uint32\s(\d)+`)
var reClean = regexp.MustCompile(`(string|uint32|\")+`)

// Parse the variant output from DBus into map of string
func cleanDBusOutput(output string) map[string]string {
	outputMap := make(map[string]string, 0)

	// Start regex replacing for important values
	s := re.ReplaceAllString(output, "")
	outputArray := reFind.FindAllString(s, -1)

	if outputArray != nil {
		var key string
		var invert = 0
		// The regex should cut things down to an alternating key:value after being trimmed
		// We add these to the map, and add a "Meta" key when it would normally be empty (as the first in the array)
		for i, value := range outputArray {
			newValue := strings.TrimSpace(reClean.ReplaceAllString(value, ""))
			// Some devices have this meta value as the first entry (iOS mainly)
			// we should swap key/value pairs if so
			if i == 0 && (newValue == "Item" || newValue == "playing" || newValue == "paused") {
				invert = 1
				key = "Meta"
			}

			// Define key or insert into map if defined
			if i%2 == invert {
				key = newValue
			} else {
				outputMap[key] = newValue
			}
		}

	} else {
		BluetoothStatus.Log(status.Error(), "Error parsing dbus output")
	}

	return outputMap
}

// EnableAutoRefresh continously refreshes bluetooth media devices
func EnableAutoRefresh() {
	BluetoothStatus.Log(status.OK(), "Enabling auto refresh of BT address")
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
	BluetoothStatus.Log(status.OK(), "Forcing refresh of BT address")
	go getConnectedAddress()
}

// getConnectedAddress will find and replace the playing media device
// this should be run continuously to check for changes in connection
func getConnectedAddress() string {
	args := "busctl tree org.bluez | grep /fd | head -n 1 | sed -n 's/.*\\/org\\/bluez\\/hci0\\/dev_\\(.*\\)\\/.*/\\1/p'"
	out, err := exec.Command("bash", "-c", args).Output()

	if err != nil {
		BluetoothStatus.Log(status.Error(), err.Error())
		return err.Error()
	}

	// Use new device if found
	newAddress := strings.TrimSpace(string(out))
	if newAddress != "" && btAddress != newAddress {
		BluetoothStatus.Log(status.OK(), "Found new connected media device with address: "+newAddress)
		SetAddress(newAddress)
	}

	return string(out)
}

// SetAddress makes address given in args available to all dbus functions
func SetAddress(address string) {
	// Format address for dbus
	if address != "" {
		btAddress = strings.Replace(strings.TrimSpace(address), ":", "_", -1)
		BluetoothStatus.Log(status.OK(), "Now routing Bluetooth commands to "+btAddress)

		// Set new address to persist in settings file
		settings.SetSetting("CONFIG", "BLUETOOTH_ADDRESS", btAddress)
	}
}

// SendDBusCommand used as a general BT control function for these endpoints
func SendDBusCommand(args []string, hideOutput bool) (string, bool) {
	if btAddress != "" {
		// Fill in the meta nonsense
		args = append([]string{"--system", "--print-reply", "--type=method_call", "--dest=org.bluez"}, args...)
		out, err := exec.Command("dbus-send", args...).Output()

		if err != nil {
			BluetoothStatus.Log(status.Error(), err.Error())
			return err.Error(), false
		}

		if !hideOutput {
			BluetoothStatus.Log(status.OK(), string(out))
		}

		return string(out), true
	}

	BluetoothStatus.Log(status.Warning(), "No valid BT Address to run command")

	return "No valid BT Address to run command", false
}

// Connect bluetooth device
func Connect(w http.ResponseWriter, r *http.Request) {
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + btAddress, "org.bluez.Device1.Connect"}, false)
	json.NewEncoder(w).Encode("OK")
}

// Disconnect bluetooth device
func Disconnect(w http.ResponseWriter, r *http.Request) {
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + btAddress, "org.bluez.Device1.Disconnect"}, false)
	json.NewEncoder(w).Encode("OK")
}

// GetDeviceInfo attempts to get metadata about connected device
func GetDeviceInfo(w http.ResponseWriter, r *http.Request) {
	BluetoothStatus.Log(status.OK(), "Getting device info...")
	result, ok := SendDBusCommand([]string{"/org/bluez/hci0/dev_" + btAddress + "/player0", "org.freedesktop.DBus.Properties.Get", "string:org.bluez.MediaPlayer1", "string:Status"}, true)
	if ok {
		json.NewEncoder(w).Encode(cleanDBusOutput(result))
	} else {
		json.NewEncoder(w).Encode("Error")
	}
}

// GetMediaInfo attempts to get metadata about current track
func GetMediaInfo(w http.ResponseWriter, r *http.Request) {
	BluetoothStatus.Log(status.OK(), "Getting device info...")
	result, ok := SendDBusCommand([]string{"/org/bluez/hci0/dev_" + btAddress + "/player0", "org.freedesktop.DBus.Properties.Get", "string:org.bluez.MediaPlayer1", "string:Status"}, true)
	if ok {
		deviceStatus := cleanDBusOutput(result)

		BluetoothStatus.Log(status.OK(), "Getting media info...")
		result, ok := SendDBusCommand([]string{"/org/bluez/hci0/dev_" + btAddress + "/player0", "org.freedesktop.DBus.Properties.Get", "string:org.bluez.MediaPlayer1", "string:Track"}, true)
		if ok {
			// Append device status to media info
			cleanResult := cleanDBusOutput(result)
			cleanResult["Status"] = deviceStatus["Meta"]

			// Append Album / Artwork slug if both exist
			album, albumOK := cleanResult["Album"]
			artist, artistOK := cleanResult["Artist"]
			if albumOK && artistOK {
				cleanResult["Album_Artwork"] = slug.Make(artist) + "/" + slug.Make(album) + ".jpg"
			}

			// Echo back all info
			json.NewEncoder(w).Encode(cleanResult)
		} else {
			json.NewEncoder(w).Encode("Error")
		}
	} else {
		json.NewEncoder(w).Encode("Error")
	}
}

// Prev skips to previous track
func Prev(w http.ResponseWriter, r *http.Request) {
	BluetoothStatus.Log(status.OK(), "Going to previous track...")
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + btAddress + "/player0", "org.bluez.MediaPlayer1.Previous"}, false)
	json.NewEncoder(w).Encode("OK")
}

// Next skips to next track
func Next(w http.ResponseWriter, r *http.Request) {
	BluetoothStatus.Log(status.OK(), "Going to next track...")
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + btAddress + "/player0", "org.bluez.MediaPlayer1.Next"}, false)
	json.NewEncoder(w).Encode("OK")
}

// Play attempts to play bluetooth media
func Play(w http.ResponseWriter, r *http.Request) {
	BluetoothStatus.Log(status.OK(), "Attempting to play media...")
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + btAddress + "/player0", "org.bluez.MediaPlayer1.Play"}, false)
	json.NewEncoder(w).Encode("OK")
}

// Pause attempts to pause bluetooth media
func Pause(w http.ResponseWriter, r *http.Request) {
	BluetoothStatus.Log(status.OK(), "Attempting to pause media...")
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + btAddress + "/player0", "org.bluez.MediaPlayer1.Pause"}, false)
	json.NewEncoder(w).Encode("OK")
}
