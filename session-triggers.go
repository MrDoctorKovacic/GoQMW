//
// This file contains modifier functions for the main session defined in session.go
// These take a POSTed value and start triggers or make adjustments
//
// Most here are specific to my setup only, and likely not generalized
//
package main

import (
	"fmt"
	"strconv"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
)

type triggerFunction func(triggerPackage *SessionPackage)

// mapping of session names to trigger functions
var triggers map[string]triggerFunction

func initTriggers() {
	triggers = map[string]triggerFunction{
		"AUX_VOLTAGE_RAW":     tAuxVoltage,
		"AUX_CURRENT_RAW":     tAuxCurrent,
		"KEY_POSITION":        tKeyPosition,
		"LIGHT_SENSOR_REASON": tLightSensorReason,
	}
	SessionStatus.Log(logging.OK(), "Initialized triggers")
}

// Process session values by combining or otherwise modifying once posted
func (triggerPackage *SessionPackage) processSessionTriggers() {
	if Config.VerboseOutput {
		SessionStatus.Log(logging.OK(), fmt.Sprintf("Triggered post processing for session name %s", triggerPackage.Name))
	}

	// Pull trigger function from mapping above
	trigger, ok := triggers[triggerPackage.Name]

	if Config.VerboseOutput && !ok {
		SessionStatus.Log(logging.Error(), fmt.Sprintf("Trigger mapping for %s does not exist, skipping", triggerPackage.Name))
		return
	}

	// Trigger the function
	trigger(triggerPackage)
}

//
// From here on out are the trigger functions.
// We're taking actions based on the values or a combination of values
// from the session.
//

// Resistance values and modifiers to the incoming Voltage sensor value
func tAuxVoltage(triggerPackage *SessionPackage) {
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
	SetSessionRawValue("AUX_VOLTAGE", fmt.Sprintf("%.3f", realVoltage))
	SetSessionRawValue("AUX_VOLTAGE_MODIFIER", fmt.Sprintf("%.3f", voltageModifier))
}

// Modifiers to the incoming Current sensor value
func tAuxCurrent(triggerPackage *SessionPackage) {
	currentFloat, err := strconv.ParseFloat(triggerPackage.Data.Value, 64)

	if err != nil {
		SessionStatus.Log(logging.Error(), fmt.Sprintf("Failed to convert string %s to float", triggerPackage.Data.Value))
		return
	}

	realCurrent := 2000 * ((((currentFloat * 3.3) / 4095.0) - 1.5) / 185)
	SetSessionRawValue("AUX_CURRENT", fmt.Sprintf("%.3f", realCurrent))
}

// Trigger for booting boards/tablets
// TODO: Smarter shutdown timings? After 10 mins?
func tKeyPosition(triggerPackage *SessionPackage) {
	switch triggerPackage.Data.Value {
	case "POS_1":
		WriteSerial("powerOnBoard")
		WriteSerial("powerOnTablet")
	case "OFF":
		// Start board shutdown
		WriteSerial("powerOffBoard")
		WriteSerial("powerOffTablet")
	}
}

// Alert me when it's raining and windows are up
func tLightSensorReason(triggerPackage *SessionPackage) {
	keyPosition, err1 := GetSessionValue("KEY_POSITION")
	doorsLocked, err2 := GetSessionValue("DOORS_LOCKED")
	delta, err3 := formatting.CompareTimeToNow(doorsLocked.LastUpdate, Timezone)

	if err1 == nil && err2 == nil && err3 == nil {
		if triggerPackage.Data.Value == "RAIN" &&
			keyPosition.Value == "OFF" &&
			doorsLocked.Value == "TRUE" &&
			delta.Minutes() > 5 {
			// TODO: ALERT ME HERE
		}
	}
}
