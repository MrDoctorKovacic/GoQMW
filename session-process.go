//
// This file contains modifier functions for the main session defined in session.go
// These take a POSTed value and make adjustments depending on the name / value
// Most here are specific to my setup only, and likely not generalized
//
package main

import (
	"fmt"
	"strconv"

	"github.com/MrDoctorKovacic/MDroid-Core/logging"
)

// Process session values by combining or otherwise modifying once posted
func (triggerPackage *SessionPackage) postProcessSession() {
	SessionStatus.Log(logging.OK(), fmt.Sprintf("Triggered post processing for session name %s", triggerPackage.Name))
	switch triggerPackage.Name {
	case "AUX_VOLTAGE_RAW":
		triggerPackage.modifyAuxVoltage()
	case "AUX_CURRENT_RAW":
		triggerPackage.modifyAuxCurrent()
	}
}

// Resistance values and modifiers to the incoming Voltage sensor value
func (triggerPackage SessionPackage) modifyAuxVoltage() {
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
	SetSessionRawValue("AUX_VOLTAGE", strconv.FormatFloat(realVoltage, 'f', -1, 64))
	SetSessionRawValue("AUX_VOLTAGE_MODIFIER", strconv.FormatFloat(voltageModifier, 'f', -1, 64))
}

// Resistance values and modifiers to the incoming Current sensor value
func (triggerPackage SessionPackage) modifyAuxCurrent() {
	currentFloat, err := strconv.ParseFloat(triggerPackage.Data.Value, 64)

	if err != nil {
		SessionStatus.Log(logging.Error(), fmt.Sprintf("Failed to convert string %s to float", triggerPackage.Data.Value))
		return
	}

	realCurrent := 2000 * ((((currentFloat * 3.3) / 4095.0) - 1.5) / 185)
	SetSessionRawValue("AUX_CURRENT", strconv.FormatFloat(realCurrent, 'f', -1, 64))
}
