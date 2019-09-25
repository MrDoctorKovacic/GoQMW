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
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
	"github.com/MrDoctorKovacic/MDroid-Core/pybus"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
)

// Process session values by combining or otherwise modifying once posted
func (session *Session) processSessionTriggers(triggerPackage *sessionPackage) {
	if session.Config.VerboseOutput {
		SessionStatus.Log(logging.OK(), fmt.Sprintf("Triggered post processing for session name %s", triggerPackage.Name))
	}

	// Pull trigger function
	switch triggerPackage.Name {
	case "MAIN_VOLTAGE_RAW":
		session.tMainVoltage(triggerPackage)
	case "AUX_VOLTAGE_RAW":
		session.tAuxVoltage(triggerPackage)
	case "AUX_CURRENT_RAW":
		session.tAuxCurrent(triggerPackage)
	case "ACC_POWER":
		session.tAccPower(triggerPackage)
	case "LIGHT_SENSOR_REASON":
		session.tLightSensorReason(triggerPackage)
	case "SEAT_MEMORY_1":
		fallthrough
	case "SEAT_MEMORY_2":
		fallthrough
	case "SEAT_MEMORY_3":
		session.tSeatMemory(triggerPackage)
	default:
		if session.Config.VerboseOutput {
			SessionStatus.Log(logging.Error(), fmt.Sprintf("Trigger mapping for %s does not exist, skipping", triggerPackage.Name))
			return
		}
	}
}

// RepeatCommand endlessly, helps with request functions
func (session *Session) RepeatCommand(command string, sleepSeconds int) {
	for {
		// Only push repeated pybus commands when powered, otherwise the car won't sleep
		hasPower, err := session.GetSessionValue("ACC_POWER")
		if err == nil && hasPower.Value == "TRUE" {
			pybus.PushQueue(command)
		}
		time.Sleep(time.Duration(sleepSeconds) * time.Second)
	}
}

//
// From here on out are the trigger functions.
// We're taking actions based on the values or a combination of values
// from the session.
//

// Convert main raw voltage into an actual number
func (session *Session) tMainVoltage(triggerPackage *sessionPackage) {
	voltageFloat, err := strconv.ParseFloat(triggerPackage.Data.Value, 64)
	if err != nil {
		SessionStatus.Log(logging.Error(), fmt.Sprintf("Failed to convert string %s to float", triggerPackage.Data.Value))
		return
	}

	session.CreateSessionValue("MAIN_VOLTAGE", fmt.Sprintf("%.3f", (voltageFloat/1024)*24.4))
}

// Resistance values and modifiers to the incoming Voltage sensor value
func (session *Session) tAuxVoltage(triggerPackage *sessionPackage) {
	voltageFloat, err := strconv.ParseFloat(triggerPackage.Data.Value, 64)

	if err != nil {
		SessionStatus.Log(logging.Error(), fmt.Sprintf("Failed to convert string %s to float", triggerPackage.Data.Value))
		return
	}

	voltageModifier := 1.08
	if voltageFloat < 2850.0 && voltageFloat > 2700.0 {
		voltageModifier = 1.12
	} else if voltageFloat >= 2850.0 {
		voltageModifier = 1.07
	} else if voltageFloat <= 2700.0 {
		voltageModifier = 1.08
	}

	realVoltage := voltageModifier * (((voltageFloat * 3.3) / 4095.0) / 0.2)
	session.CreateSessionValue("AUX_VOLTAGE", fmt.Sprintf("%.3f", realVoltage))
	session.CreateSessionValue("AUX_VOLTAGE_MODIFIER", fmt.Sprintf("%.3f", voltageModifier))

	sentPowerWarning, err := session.GetSessionValue("SENT_POWER_WARNING")
	if err != nil {
		session.CreateSessionValue("SENT_POWER_WARNING", "FALSE")
	}

	// SHUTDOWN the system if voltage is below 11.3 to preserve our battery
	// TODO: right now poweroff doesn't do crap, still drains battery
	if realVoltage < 11.3 {
		if err == nil && sentPowerWarning.Value == "FALSE" {
			logging.SlackAlert(session.Config.SlackURL, fmt.Sprintf("MDROID SHUTTING DOWN! Voltage is %f (%fV)", voltageFloat, realVoltage))
			session.CreateSessionValue("SENT_POWER_WARNING", "TRUE")
		}
		//exec.Command("poweroff", "now")
	}
}

// Modifiers to the incoming Current sensor value
func (session *Session) tAuxCurrent(triggerPackage *sessionPackage) {
	currentFloat, err := strconv.ParseFloat(triggerPackage.Data.Value, 64)

	if err != nil {
		SessionStatus.Log(logging.Error(), fmt.Sprintf("Failed to convert string %s to float", triggerPackage.Data.Value))
		return
	}

	realCurrent := math.Abs(1000 * ((((currentFloat * 3.3) / 4095.0) - 1.5) / 185))
	session.CreateSessionValue("AUX_CURRENT", fmt.Sprintf("%.3f", realCurrent))
}

// Trigger for booting boards/tablets
// TODO: Smarter shutdown timings? After 10 mins?
func (session *Session) tAccPower(triggerPackage *sessionPackage) {
	// Pull needed values for power logic
	wirelessPoweredOn, _ := session.GetSessionValue("WIRELESS_POWER")
	wifiAvailable, _ := session.GetSessionValue("WIFI_CONNECTED")
	boardPoweredOn, _ := session.GetSessionValue("BOARD_POWER")
	tabletPoweredOn, _ := session.GetSessionValue("TABLET_POWER")
	raynorTargetPower, rerr := settings.Get("RAYNOR", "POWER")
	lucioTargetPower, aerr := settings.Get("LUCIO", "POWER")
	brightwingTargetPower, berr := settings.Get("BRIGHTWING", "POWER")

	// Read the target action based on current ACC Power value
	var targetAction string
	if triggerPackage.Data.Value == "TRUE" {
		targetAction = "On"
	} else if triggerPackage.Data.Value == "FALSE" {
		targetAction = "Off"
	} else {
		SessionStatus.Log(logging.Error(), fmt.Sprintf("ACC Power Trigger unexpected value: %s", triggerPackage.Data.Value))
		return
	}

	// Handle wireless power control
	if berr == nil {
		if (brightwingTargetPower == "AUTO" && wifiAvailable.Value != "TRUE" && wirelessPoweredOn.Value != "TRUE") ||
			(brightwingTargetPower == "ON" && wirelessPoweredOn.Value == "FALSE") {
			mserial.WriteSerial(session.Config.SerialControlDevice, "powerOnWireless")
		} else if brightwingTargetPower == "AUTO" && (wirelessPoweredOn.Value != triggerPackage.Data.Value) {
			mserial.WriteSerial(session.Config.SerialControlDevice, fmt.Sprintf("power%sWireless", targetAction))
		} else if brightwingTargetPower == "OFF" && wirelessPoweredOn.Value == "TRUE" {
			go mserial.MachineShutdown(session.Config.SerialControlDevice, "brightwing", time.Second*10, "powerOffWireless")
		}
	} else {
		SessionStatus.Log(logging.Error(), fmt.Sprintf("Setting read error for Brightwing. Resetting to AUTO\n%s,", berr))
		settings.Set("BRIGHTWING", "POWER", "AUTO")
	}

	// Handle video server power control
	if aerr == nil {
		if lucioTargetPower == "AUTO" && boardPoweredOn.Value != triggerPackage.Data.Value {
			mserial.WriteSerial(session.Config.SerialControlDevice, fmt.Sprintf("power%sBoard", targetAction))
		} else if lucioTargetPower == "OFF" && boardPoweredOn.Value == "TRUE" {
			mserial.CommandNetworkMachine("etc", "shutdown")
			go mserial.MachineShutdown(session.Config.SerialControlDevice, "lucio", time.Second*10, "powerOffBoard")
		} else if lucioTargetPower == "ON" && boardPoweredOn.Value == "FALSE" {
			mserial.WriteSerial(session.Config.SerialControlDevice, "powerOnBoard")
		}
	} else {
		SessionStatus.Log(logging.Error(), fmt.Sprintf("Setting read error for Artanis. Resetting to AUTO\n%s,", berr))
		settings.Set("LUCIO", "POWER", "AUTO")
	}

	// Handle tablet power control
	if rerr == nil {
		if raynorTargetPower == "AUTO" && tabletPoweredOn.Value != triggerPackage.Data.Value {
			mserial.WriteSerial(session.Config.SerialControlDevice, fmt.Sprintf("power%sTablet", targetAction))
		} else if raynorTargetPower == "OFF" && tabletPoweredOn.Value == "TRUE" {
			mserial.WriteSerial(session.Config.SerialControlDevice, "powerOffTablet")
		} else if raynorTargetPower == "ON" && tabletPoweredOn.Value == "FALSE" {
			mserial.WriteSerial(session.Config.SerialControlDevice, "powerOnTablet")
		}
	} else {
		SessionStatus.Log(logging.Error(), fmt.Sprintf("Setting read error for Raynor. Resetting to AUTO\n%s,", berr))
		settings.Set("RAYNOR", "POWER", "AUTO")
	}
}

// Alert me when it's raining and windows are down
func (session *Session) tLightSensorReason(triggerPackage *sessionPackage) {
	keyPosition, err1 := session.GetSessionValue("KEY_POSITION")
	doorsLocked, err2 := session.GetSessionValue("DOORS_LOCKED")
	windowsOpen, err2 := session.GetSessionValue("WINDOWS_OPEN")
	delta, err3 := formatting.CompareTimeToNow(doorsLocked.LastUpdate, session.Config.Timezone)

	if err1 == nil && err2 == nil && err3 == nil {
		if triggerPackage.Data.Value == "RAIN" &&
			keyPosition.Value == "OFF" &&
			doorsLocked.Value == "TRUE" &&
			windowsOpen.Value == "TRUE" &&
			delta.Minutes() > 5 {
			logging.SlackAlert(session.Config.SlackURL, "Windows are down in the rain, eh?")
		}
	}
}

// Restart different machines when seat memory buttons are pressed
func (session *Session) tSeatMemory(triggerPackage *sessionPackage) {
	switch triggerPackage.Name {
	case "SEAT_MEMORY_1":
		mserial.CommandNetworkMachine("LUCIO", "restart")
	case "SEAT_MEMORY_2":
		mserial.CommandNetworkMachine("BRIGHTWING", "restart")
	case "SEAT_MEMORY_3":
		mserial.CommandNetworkMachine("MDROID", "restart")
	}
}
