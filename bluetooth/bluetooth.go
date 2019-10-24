// Package bluetooth is a rudimentary interface between MDroid-Core and underlying BT dbus
package bluetooth

import (
	"bytes"
	"fmt"
	"net/http"
	"os/exec"
	"os/user"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/format"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/gosimple/slug"
	"github.com/rs/zerolog/log"
)

// Regex expressions for parsing dbus output
var (
	BluetoothAddress   string
	BluetoothConnected bool
	replySerialRegex   *regexp.Regexp
	findStringRegex    *regexp.Regexp
	cleanRegex         *regexp.Regexp
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

// ScanOn turns on bluetooth scan with bluetoothctl
func ScanOn() {
	var stderr bytes.Buffer
	var out bytes.Buffer
	cmd := exec.Command("echo", "scan", "on", "|", "bluetoothctl")
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Error().Msg("Error turning scan on")
		log.Error().Msg(err.Error())
		log.Error().Msg(stderr.String())
	}
}

// Parse the variant output from DBus into map of string
func cleanDBusOutput(output string) map[string]string {
	outputMap := make(map[string]string, 0)

	// Start regex replacing for important values
	s := replySerialRegex.ReplaceAllString(output, "")
	outputArray := findStringRegex.FindAllString(s, -1)

	if outputArray == nil {
		log.Error().Msg("Error parsing dbus output. Full output:")
		log.Error().Msg(output)
	}

	var (
		key    string
		invert = 0
	)
	// The regex should cut things down to an alternating key:value after being trimmed
	// We add these to the map, and add a "Meta" key when it would normally be empty (as the first in the array)
	for i, value := range outputArray {
		newValue := strings.TrimSpace(cleanRegex.ReplaceAllString(value, ""))
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

	return outputMap
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

// getConnectedAddress will find and replace the playing media device
// this should be run continuously to check for changes in connection
func getConnectedAddress() string {
	args := "busctl tree org.bluez | grep /fd | head -n 1 | sed -n 's/.*\\/org\\/bluez\\/hci0\\/dev_\\(.*\\)\\/.*/\\1/p'"
	out, err := exec.Command("bash", "-c", args).Output()

	if err != nil {
		log.Error().Msg(err.Error())
		return err.Error()
	}

	// Use new device if found
	newAddress := strings.TrimSpace(string(out))
	if newAddress != "" && BluetoothAddress != newAddress {
		log.Info().Msg("Found new connected media device with address: " + newAddress)
		SetAddress(newAddress)
	}

	return string(out)
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

// SendDBusCommand used as a general BT control function for these endpoints
func SendDBusCommand(runAs *user.User, args []string, hideOutput bool, skipAddressCheck bool) (string, bool) {
	if !skipAddressCheck && BluetoothAddress == "" {
		log.Warn().Msg("No valid BT Address to run command")
		return "No valid BT Address to run command", false
	}

	// Use current (presumably root) user if is nil
	if runAs == nil {
		var err error
		runAs, err = user.Current()
		if err != nil {
			log.Error().Msg("Error getting current user permissions in exec call")
			log.Error().Msg(err.Error())
			return "", false
		}
	}

	// Get user details
	u := *runAs
	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		log.Error().Msg("Error parsing uid into uint32")
		log.Error().Msg(err.Error())
		return "", false
	}
	gid, err := strconv.ParseUint(u.Gid, 10, 32)
	if err != nil {
		log.Error().Msg("Error parsing gid into uint32")
		log.Error().Msg(err.Error())
		return "", false
	}

	// Fill in the meta nonsense
	args = append([]string{"--system", "--type=method_call", "--dest=org.bluez", "--print-reply"}, args...)

	// Execute the build dbus command
	var stderr bytes.Buffer
	var out bytes.Buffer
	cmd := exec.Command("dbus-send", args...)
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}

	if err := cmd.Run(); err != nil {
		log.Error().Msg(err.Error())
		log.Error().Msg(stderr.String())
		log.Error().Msg("Args: " + strings.Join(args, " "))
		log.Error().Msg("Ran as: " + runAs.Username)
		return stderr.String(), false
	}

	if !hideOutput {
		log.Debug().Msg(out.String())
	}

	return out.String(), true
}

// Connect bluetooth device
func Connect(w http.ResponseWriter, r *http.Request) {
	// Turn scanning on
	ScanOn()

	log.Info().Msg("Connecting to bluetooth device...")

	/*runAs, err := user.Lookup("casey")
	if err != nil {
		log.Error().Msg("Could not lookup user")
		log.Error().Msg(err.Error())
		format.WriteResponse(&w, r, format.JSONResponse{Output: "Could not lookup user", OK: false})
		return
	}*/

	go SendDBusCommand(
		nil,
		[]string{"/org/bluez/hci0/dev_" + BluetoothAddress, "org.bluez.Device1.Connect"},
		false,
		true)

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

	runAs, err := user.Lookup("casey")
	if err != nil {
		log.Error().Msg("Could not lookup user")
		return fmt.Errorf("Could not lookup user")
	}

	SendDBusCommand(
		runAs,
		[]string{"/org/bluez/hci0/dev_" + BluetoothAddress,
			"org.bluez.Device1.Disonnect"},
		false,
		true)

	return nil
}

func askDeviceInfo() map[string]string {
	log.Info().Msg("Getting device info...")

	deviceMessage := []string{"/org/bluez/hci0/dev_" + BluetoothAddress + "/player0", "org.freedesktop.DBus.Properties.Get", "string:org.bluez.MediaPlayer1", "string:Status"}
	result, ok := SendDBusCommand(nil, deviceMessage, true, false)
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
	result, ok := SendDBusCommand(nil, mediaMessage, true, false)
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
	go SendDBusCommand(nil, []string{"/org/bluez/hci0/dev_" + BluetoothAddress + "/player0", "org.bluez.MediaPlayer1.Previous"}, false, false)
	format.WriteResponse(&w, r, format.JSONResponse{Output: "OK", OK: true})
}

// Next skips to next track
func Next(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Going to next track...")
	go SendDBusCommand(nil, []string{"/org/bluez/hci0/dev_" + BluetoothAddress + "/player0", "org.bluez.MediaPlayer1.Next"}, false, false)
	format.WriteResponse(&w, r, format.JSONResponse{Output: "OK", OK: true})
}

// Play attempts to play bluetooth media
func Play(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Attempting to play media...")
	go SendDBusCommand(nil, []string{"/org/bluez/hci0/dev_" + BluetoothAddress + "/player0", "org.bluez.MediaPlayer1.Play"}, false, false)
	format.WriteResponse(&w, r, format.JSONResponse{Output: "OK", OK: true})
}

// Pause attempts to pause bluetooth media
func Pause(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Attempting to pause media...")
	go SendDBusCommand(nil, []string{"/org/bluez/hci0/dev_" + BluetoothAddress + "/player0", "org.bluez.MediaPlayer1.Pause"}, false, false)
	format.WriteResponse(&w, r, format.JSONResponse{Output: "OK", OK: true})
}
