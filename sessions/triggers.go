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

// Process session values by combining or otherwise modifying once posted
func processSessionTriggers(triggerPackage sessionPackage) {
	if settings.Config.VerboseOutput {
		status.Log(logging.OK(), fmt.Sprintf("Triggered post processing for session name %s", triggerPackage.Name))
	}

	// Pull trigger function
	switch triggerPackage.Name {
	case "MAIN_VOLTAGE_RAW", "AUX_VOLTAGE_RAW":
		tVoltage(&triggerPackage)
	case "AUX_CURRENT_RAW":
		tAuxCurrent(&triggerPackage)
	case "ACC_POWER":
		tAccPower(&triggerPackage)
	case "KEY_STATE":
		tKeyState(&triggerPackage)
	case "WIRELESS_POWER":
		tLTEOn(&triggerPackage)
	case "LIGHT_SENSOR_REASON":
		tLightSensorReason(&triggerPackage)
	case "SEAT_MEMORY_1", "SEAT_MEMORY_2", "SEAT_MEMORY_3":
		tSeatMemory(&triggerPackage)
	default:
		if settings.Config.VerboseOutput {
			status.Log(logging.Error(), fmt.Sprintf("Trigger mapping for %s does not exist, skipping", triggerPackage.Name))
			return
		}
	}
}

//
// From here on out are the trigger functions.
// We're taking actions based on the values or a combination of values
// from the session.
//

// Convert main raw voltage into an actual number
func tVoltage(triggerPackage *sessionPackage) {
	voltageFloat, err := strconv.ParseFloat(triggerPackage.Data.Value, 64)
	if err != nil {
		status.Log(logging.Error(), fmt.Sprintf("Failed to convert string %s to float", triggerPackage.Data.Value))
		return
	}

	SetValue(triggerPackage.Name[0:len(triggerPackage.Name)-4], fmt.Sprintf("%.3f", (voltageFloat/1024)*24.4))
}

// Modifiers to the incoming Current sensor value
func tAuxCurrent(triggerPackage *sessionPackage) {
	currentFloat, err := strconv.ParseFloat(triggerPackage.Data.Value, 64)

	if err != nil {
		status.Log(logging.Error(), fmt.Sprintf("Failed to convert string %s to float", triggerPackage.Data.Value))
		return
	}

	realCurrent := math.Abs(1000 * ((((currentFloat * 3.3) / 4095.0) - 1.5) / 185))
	SetValue("AUX_CURRENT", fmt.Sprintf("%.3f", realCurrent))
}

// Trigger for booting boards/tablets
func tAccPower(triggerPackage *sessionPackage) {
	// Read the target action based on current ACC Power value
	var (
		accOn    bool
		wireless = power{settingComp: "BRIGHTWING", settingName: "POWER"}
		wifi     = power{settingComp: "", settingName: ""}
		angel    = power{settingComp: "VARIAN", settingName: "ANGEL_EYES"}
		tablet   = power{settingComp: "RAYNOR", settingName: "POWER"}
		board    = power{settingComp: "LUCIO", settingName: "POWER"}
	)

	// Check incoming ACC power value is valid
	switch triggerPackage.Data.Value {
	case "TRUE":
		accOn = true
	case "FALSE":
		accOn = false
	default:
		status.Log(logging.Error(), fmt.Sprintf("ACC Power Trigger unexpected value: %s", triggerPackage.Data.Value))
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

	// Add angel eyes, if they're set to be on
	if angel.powerTarget != "AUTO" || angel.on {
		modules["Angel"] = angel
	}

	for name, module := range modules {
		genericPowerTrigger(accOn, name, module)
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
		status.Log(logging.Error(), fmt.Sprintf("Setting read error for %s. Resetting to AUTO\n%s\n%s,", name, module.errOn.Error(), module.errTarget.Error()))
		if module.settingComp != "" && module.settingName != "" {
			settings.Set(module.settingComp, module.settingName, "AUTO")
		}
	}
}

func tKeyState(triggerPackage *sessionPackage) {
	angel := power{settingComp: "VARIAN", settingName: "ANGEL_EYES"}
	angel.on, angel.errOn = GetBool("ANGEL_EYES_POWER")
	angel.powerTarget, angel.errTarget = settings.Get(angel.settingComp, angel.settingName)

	shouldBeTriggered := triggerPackage.Data.Value != "FALSE" && triggerPackage.Data.Value != ""

	// Pass angel module to generic power trigger
	genericPowerTrigger(shouldBeTriggered, "Angel", angel)
}

func tLTEOn(triggerPackage *sessionPackage) {
	lteOn, err := Get("WIRELESS_POWER")
	if err != nil {
		status.Log(logging.Error(), err.Error())
		return
	}

	if triggerPackage.Data.Value == "FALSE" && lteOn.Value == "TRUE" {
		// When board is turned off but doesn't have time to reflect LTE status
		SetValue("LTE_ON", "FALSE")
	}
}

// Alert me when it's raining and windows are down
func tLightSensorReason(triggerPackage *sessionPackage) {
	keyPosition, _ := Get("KEY_POSITION")
	doorsLocked, _ := Get("DOORS_LOCKED")
	windowsOpen, _ := Get("WINDOWS_OPEN")
	delta, err := formatting.CompareTimeToNow(doorsLocked.LastUpdate, gps.GetTimezone())

	if err != nil {
		if triggerPackage.Data.Value == "RAIN" &&
			keyPosition.Value == "OFF" &&
			doorsLocked.Value == "TRUE" &&
			windowsOpen.Value == "TRUE" &&
			delta.Minutes() > 5 {
			logging.SlackAlert(settings.Config.SlackURL, "Windows are down in the rain, eh?")
		}
	}
}

// Restart different machines when seat memory buttons are pressed
func tSeatMemory(triggerPackage *sessionPackage) {
	switch triggerPackage.Name {
	case "SEAT_MEMORY_1":
		mserial.CommandNetworkMachine("LUCIO", "restart")
	case "SEAT_MEMORY_2":
		mserial.CommandNetworkMachine("BRIGHTWING", "restart")
	case "SEAT_MEMORY_3":
		mserial.CommandNetworkMachine("MDROID", "restart")
	}
}
