package sessions

//
// This file contains modifier functions for the main session defined in session.go
// These take a POSTed value and start triggers or make adjustments
//
// Most here are specific to my setup only, and likely not generalized
//

import (
	"fmt"
	"math"
	"strconv"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/gps"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
)

// Define temporary holding struct for power values
type power struct {
	on          bool
	powerTarget string
	errOn       error
	errTarget   error
	triggerOn   bool
	settingComp string
	settingName string
}

// Read the target action based on current ACC Power value
var (
	wirelessDef = power{settingComp: "LTE", settingName: "POWER"}
	wifiDef     = power{settingComp: "", settingName: ""}
	angelDef    = power{settingComp: "ANGEL_EYES", settingName: "POWER"}
	tabletDef   = power{settingComp: "TABLET", settingName: "POWER"}
	boardDef    = power{settingComp: "BOARD", settingName: "POWER"}
)

// Process session values by combining or otherwise modifying once posted
func processSessionTriggers(hook sessionPackage) {
	status.Log(logging.Debug(), fmt.Sprintf("Triggered post processing for session name %s", hook.Name))

	// Pull trigger function
	switch hook.Name {
	case "MAIN_VOLTAGE_RAW", "AUX_VOLTAGE_RAW":
		voltage(&hook)
	case "AUX_CURRENT_RAW":
		auxCurrent(&hook)
	case "ACC_POWER":
		accPower(&hook)
	case "KEY_STATE":
		keyState(&hook)
	case "WIRELESS_POWER":
		lteOn(&hook)
	case "LIGHT_SENSOR_REASON":
		lightSensorReason(&hook)
	case "LIGHT_SENSOR_ON":
		lightSensorOn(&hook)
	case "SEAT_MEMORY_1", "SEAT_MEMORY_2", "SEAT_MEMORY_3":
		seatMemory(&hook)
	default:
		status.Log(logging.Debug(), fmt.Sprintf("Trigger mapping for %s does not exist, skipping", hook.Name))
	}
}

//
// From here on out are the trigger functions.
// We're taking actions based on the values or a combination of values
// from the session.
//

// Convert main raw voltage into an actual number
func voltage(hook *sessionPackage) {
	voltageFloat, err := strconv.ParseFloat(hook.Data.Value, 64)
	if err != nil {
		status.Log(logging.Error(), fmt.Sprintf("Failed to convert string %s to float", hook.Data.Value))
		return
	}

	SetValue(hook.Name[0:len(hook.Name)-4], fmt.Sprintf("%.3f", (voltageFloat/1024)*24.4))
}

// Modifiers to the incoming Current sensor value
func auxCurrent(hook *sessionPackage) {
	currentFloat, err := strconv.ParseFloat(hook.Data.Value, 64)

	if err != nil {
		status.Log(logging.Error(), fmt.Sprintf("Failed to convert string %s to float", hook.Data.Value))
		return
	}

	realCurrent := math.Abs(1000 * ((((currentFloat * 3.3) / 4095.0) - 1.5) / 185))
	SetValue("AUX_CURRENT", fmt.Sprintf("%.3f", realCurrent))
}

// Trigger for booting boards/tablets
func accPower(hook *sessionPackage) {
	// Read the target action based on current ACC Power value
	var accOn bool
	wireless := wirelessDef
	wifi := wifiDef
	angel := angelDef
	tablet := tabletDef
	board := boardDef

	// Check incoming ACC power value is valid
	switch hook.Data.Value {
	case "TRUE":
		accOn = true
	case "FALSE":
		accOn = false
	default:
		status.Log(logging.Error(), fmt.Sprintf("ACC Power Trigger unexpected value: %s", hook.Data.Value))
		return
	}

	// Verbose, but pull all the necessary configuration data
	wireless.on, wireless.errOn = GetBool("WIRELESS_POWER")
	wireless.powerTarget, wireless.errTarget = settings.Get(wireless.settingComp, wireless.settingName)
	wifi.on, wifi.errOn = GetBool("WIFI_CONNECTED")
	tablet.on, tablet.errOn = GetBool("TABLET_POWER")
	tablet.powerTarget, tablet.errTarget = settings.Get(tablet.settingComp, tablet.settingName)
	angel.on, angel.errOn = GetBool("ANGEL_EYES_POWER")
	angel.powerTarget, angel.errTarget = settings.Get(angel.settingComp, angel.settingName)
	board.on, board.errOn = GetBool("BOARD_POWER")
	board.powerTarget, board.errTarget = settings.Get(board.settingComp, board.settingName)

	// Trigger wireless, based on wifi status
	if wireless.powerTarget == "AUTO" && !wifi.on && !wireless.on {
		wireless.powerTarget = "ON"
	}

	// Handle more generic modules
	modules := map[string]power{"Board": board, "Tablet": tablet, "Wireless": wireless}

	for name, module := range modules {
		go genericPowerTrigger(accOn, name, module)
	}
}

// Error check against module's status fetches, then check if we're powering on or off
func genericPowerTrigger(accOn bool, name string, module power) {
	if module.errOn == nil && module.errTarget == nil {
		if (module.powerTarget == "AUTO" && !module.on && accOn) || (module.powerTarget == "ON" && !module.on) {
			mserial.WriteSerial(settings.Config.SerialControlDevice, fmt.Sprintf("powerOn%s", name))
		} else if (module.powerTarget == "AUTO" && module.on && !accOn) || (module.powerTarget == "OFF" && module.on) {
			gracefulShutdown(name)
		}
	} else if module.errTarget != nil || module.errOn != nil {
		status.Log(logging.Error(), fmt.Sprintf("Setting read error for %s. Resetting to AUTO", name))
		if module.settingComp != "" && module.settingName != "" {
			settings.Set(module.settingComp, module.settingName, "AUTO")
		}
	}
}

func keyState(hook *sessionPackage) {
	angel := angelDef
	angel.on, angel.errOn = GetBool("ANGEL_EYES_POWER")
	angel.powerTarget, angel.errTarget = settings.Get(angel.settingComp, angel.settingName)

	keyIsIn := hook.Data.Value != "FALSE" && hook.Data.Value != ""

	// Pass angel module to generic power trigger
	genericPowerTrigger(keyIsIn, "Angel", angel)
}

func lteOn(hook *sessionPackage) {
	lteOn, err := Get("WIRELESS_POWER")
	if err != nil {
		status.Log(logging.Error(), err.Error())
		return
	}

	if hook.Data.Value == "FALSE" && lteOn.Value == "TRUE" {
		// When board is turned off but doesn't have time to reflect LTE status
		SetValue("LTE_ON", "FALSE")
	}
}

func lightSensorOn(hook *sessionPackage) {
	angel := angelDef
	angel.on, angel.errOn = GetBool("ANGEL_EYES_POWER")
	angel.powerTarget, angel.errTarget = settings.Get(angel.settingComp, angel.settingName)
	lightSensorOn := hook.Data.Value == "TRUE"

	// Pass angel module to generic power trigger
	genericPowerTrigger(lightSensorOn, "Angel", angel)
}

// Alert me when it's raining and windows are down
func lightSensorReason(hook *sessionPackage) {
	keyPosition, _ := Get("KEY_POSITION")
	doorsLocked, _ := Get("DOORS_LOCKED")
	windowsOpen, _ := Get("WINDOWS_OPEN")
	delta, err := formatting.CompareTimeToNow(doorsLocked.LastUpdate, gps.GetTimezone())

	if err != nil {
		if hook.Data.Value == "RAIN" &&
			keyPosition.Value == "OFF" &&
			doorsLocked.Value == "TRUE" &&
			windowsOpen.Value == "TRUE" &&
			delta.Minutes() > 5 {
			logging.SlackAlert(settings.Config.SlackURL, "Windows are down in the rain, eh?")
		}
	}
}

// Restart different machines when seat memory buttons are pressed
func seatMemory(hook *sessionPackage) {
	switch hook.Name {
	case "SEAT_MEMORY_1":
		mserial.CommandNetworkMachine("BOARD", "restart")
	case "SEAT_MEMORY_2":
		mserial.CommandNetworkMachine("LTE", "restart")
	case "SEAT_MEMORY_3":
		mserial.CommandNetworkMachine("MDROID", "restart")
	}
}
