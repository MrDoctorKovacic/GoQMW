package bluetooth

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/qcasey/MDroid-Core/sessions"
	"github.com/rs/zerolog/log"
)

type command struct {
	name string
	args []string
}

// ScanOn turns on bluetooth scan with bluetoothctl
func ScanOn() {
	log.Info().Msg("Turning scan on...")
	runCommand("tmux", []string{"kill-session", "-t", "bluetoothConnect"})

	//command{name: "tmux", args: []string{"send-keys", "-t", "bluetoothConnect", "-l", fmt.Sprintf("connect %s", BluetoothAddress)}},
	tmuxCommands := []command{
		command{name: "tmux", args: []string{"new-session", "-d", "-s", "bluetoothConnect", "bluetoothctl"}},
		command{name: "tmux", args: []string{"send-keys", "-t", "bluetoothConnect", "-l", "scan on"}},
		command{name: "tmux", args: []string{"send-keys", "-t", "bluetoothConnect", "Enter"}},
	}

	for _, c := range tmuxCommands {
		runCommand(c.name, c.args)
	}
}

func runCommand(commandName string, commandArgs []string) {
	var stderr bytes.Buffer
	var out bytes.Buffer
	cmd := exec.Command(commandName, commandArgs...)
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Error().Msg("Error turning scan on")
		log.Error().Msg(err.Error())
		log.Error().Msg(stderr.String())
		return
	}
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
	sessions.Set("connected_bluetooth_address", newAddress, false)

	return string(out)
}

// SendDBusCommand used as a general BT control function for these endpoints
func SendDBusCommand(args []string, hideOutput bool, skipAddressCheck bool) (string, bool) {
	if !skipAddressCheck && BluetoothAddress == "" {
		log.Warn().Msg("No valid BT Address to run command")
		return "No valid BT Address to run command", false
	}

	// Fill in the meta nonsense
	args = append([]string{"--system", "--type=method_call", "--dest=org.bluez", "--print-reply"}, args...)

	// Execute the build dbus command
	var stderr bytes.Buffer
	var out bytes.Buffer
	cmd := exec.Command("dbus-send", args...)
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Error().Msg(err.Error())
		log.Error().Msg(stderr.String())
		log.Error().Msg("Args: " + strings.Join(args, " "))
		return stderr.String(), false
	}

	if !hideOutput {
		log.Debug().Msg(out.String())
	}

	return out.String(), true
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
