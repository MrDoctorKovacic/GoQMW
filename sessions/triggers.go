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
	"time"

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
// TODO: Smarter shutdown timings? After 10 mins?
func tAccPower(triggerPackage *sessionPackage) {
	// Read the target action based on current ACC Power value
	var (
		targetAction string
		accOn        bool
		wireless     = power{}
		wifi         = power{}
		tablet       = power{}
		angel        = power{}
		board        = power{}
	)
	switch triggerPackage.Data.Value {
	case "TRUE":
		targetAction = "On"
		accOn = true
	case "FALSE":
		targetAction = "Off"
		accOn = false
	default:
		status.Log(logging.Error(), fmt.Sprintf("ACC Power Trigger unexpected value: %s", triggerPackage.Data.Value))
		return
	}

	// Verbose, but pull all the necessary configuration data
	wireless.on, wireless.errOn = GetBool("WIRELESS_POWER")
	wireless.powerTarget, wireless.errTarget = settings.Get("BRIGHTWING", "POWER")
	wifi.on, wifi.errOn = GetBool("WIFI_CONNECTED")
	tablet.on, tablet.errOn = GetBool("TABLET_POWER")
	tablet.powerTarget, tablet.errTarget = settings.Get("RAYNOR", "POWER")
	angel.on, angel.errOn = GetBool("ANGEL_EYES_POWER")
	angel.powerTarget, angel.errTarget = settings.Get("VARIAN", "ANGEL_EYES")
	board.on, board.errOn = GetBool("BOARD_POWER")
	board.powerTarget, board.errTarget = settings.Get("LUCIO", "POWER")

	// Trigger wireless, based on wifi status
	if wireless.powerTarget == "AUTO" && !wifi.on && !wireless.on {
		wireless.powerTarget = "ON"
	}

	// Handle more generic modules
	modules := map[string]power{"Angel": angel, "Tablet": tablet, "Wireless": wireless}
	for name, module := range modules {
		if module.errOn == nil && module.errTarget == nil {
			if module.powerTarget == "ON" && !module.on {
				mserial.WriteSerial(settings.Config.SerialControlDevice, fmt.Sprintf("powerOn%s", name))
			} else if module.powerTarget == "AUTO" && (module.on != accOn) {
				mserial.WriteSerial(settings.Config.SerialControlDevice, fmt.Sprintf("power%s%s", targetAction, name))
			} else if module.powerTarget == "OFF" && module.on {
				mserial.WriteSerial(settings.Config.SerialControlDevice, fmt.Sprintf("powerOff%s", name))
			}
		} else if module.errTarget != nil {
			status.Log(logging.Error(), fmt.Sprintf("Setting read error for %s. Resetting to AUTO\n%s\n%s,", name, module.errOn.Error(), module.errTarget.Error()))
			switch name {
			case "Angel":
				settings.Set("VARIAN", "ANGEL_EYES", "AUTO")
			case "Tablet":
				settings.Set("RAYNOR", "POWER", "AUTO")
			case "Wireless":
				settings.Set("BRIGHTWING", "POWER", "AUTO")
			}
		}
	}

	// Handle video server power control
	if board.errOn == nil && board.errTarget == nil {
		if board.powerTarget == "AUTO" && board.on != accOn {
			if !accOn {
				mserial.CommandNetworkMachine("etc", "shutdown")
				go mserial.MachineShutdown(settings.Config.SerialControlDevice, "lucio", time.Second*10, "powerOffBoard")
			} else {
				go mserial.WriteSerial(settings.Config.SerialControlDevice, fmt.Sprintf("power%sBoard", targetAction))
			}
		} else if board.powerTarget == "OFF" && board.on {
			mserial.CommandNetworkMachine("etc", "shutdown")
			go mserial.MachineShutdown(settings.Config.SerialControlDevice, "lucio", time.Second*10, "powerOffBoard")
		} else if board.powerTarget == "ON" && !board.on {
			go mserial.WriteSerial(settings.Config.SerialControlDevice, "powerOnBoard")
		}
	} else {
		status.Log(logging.Error(), fmt.Sprintf("Setting read error for Artanis. Resetting to AUTO\n%s\n%s,", board.errOn.Error(), board.errTarget.Error()))
		settings.Set("LUCIO", "POWER", "AUTO")
	}
}

func tLTEOn(triggerPackage *sessionPackage) {
	lteOn, _ := Get("WIRELESS_POWER")
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
