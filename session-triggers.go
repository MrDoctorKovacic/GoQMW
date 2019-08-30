//
// This file contains modifier functions for the main session defined in session.go
// These take a POSTed value and start triggers or make adjustments
//
// Most here are specific to my setup only, and likely not generalized
//
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/pybus"
)

// Process session values by combining or otherwise modifying once posted
func (triggerPackage *SessionPackage) processSessionTriggers() {
	if Config.VerboseOutput {
		SessionStatus.Log(logging.OK(), fmt.Sprintf("Triggered post processing for session name %s", triggerPackage.Name))
	}

	// Pull trigger function
	switch triggerPackage.Name {
	case "AUX_VOLTAGE_RAW":
		tAuxVoltage(triggerPackage)
	case "AUX_CURRENT_RAW":
		tAuxCurrent(triggerPackage)
	case "KEY_POSITION":
		tKeyPosition(triggerPackage)
	case "LIGHT_SENSOR_REASON":
		tLightSensorReason(triggerPackage)
	default:
		if Config.VerboseOutput {
			SessionStatus.Log(logging.Error(), fmt.Sprintf("Trigger mapping for %s does not exist, skipping", triggerPackage.Name))
			return
		}
	}
}

func slackAlert(message string) {
	if Config.SlackURL != "" {
		var jsonStr = []byte(fmt.Sprintf(`{"text":"%s"}`, message))
		req, _ := http.NewRequest("POST", Config.SlackURL, bytes.NewBuffer(jsonStr))
		req.Header.Set("X-Custom-Header", "myvalue")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		fmt.Println("response Status:", resp.Status)
		fmt.Println("response Headers:", resp.Header)
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("response Body:", string(body))
	}
}

// RepeatCommand endlessly, helps with request functions
func RepeatCommand(command string, sleepSeconds int) {
	for {
		// Only push repeated pybus commands when powered, otherwise the car won't sleep
		hasPower, err := GetSessionValue("ACC_POWER")
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

	sentPowerWarning, err := GetSessionValue("SENT_POWER_WARNING")
	if err != nil {
		SetSessionRawValue("SENT_POWER_WARNING", "FALSE")
	}

	// SHUTDOWN the system if voltage is below 11.3 to preserve our battery
	// TODO: right now poweroff doesn't do crap, still drains battery
	if realVoltage < 11.3 {
		if err == nil && sentPowerWarning.Value == "FALSE" {
			slackAlert(fmt.Sprintf("MDROID SHUTTING DOWN! Voltage is %f (%fV)", voltageFloat, realVoltage))
			SetSessionRawValue("SENT_POWER_WARNING", "TRUE")
		}
		//exec.Command("poweroff", "now")
	}
}

// Modifiers to the incoming Current sensor value
func tAuxCurrent(triggerPackage *SessionPackage) {
	currentFloat, err := strconv.ParseFloat(triggerPackage.Data.Value, 64)

	if err != nil {
		SessionStatus.Log(logging.Error(), fmt.Sprintf("Failed to convert string %s to float", triggerPackage.Data.Value))
		return
	}

	realCurrent := math.Abs(1000 * ((((currentFloat * 3.3) / 4095.0) - 1.5) / 185))
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
	windowsOpen, err2 := GetSessionValue("WINDOWS_OPEN")
	delta, err3 := formatting.CompareTimeToNow(doorsLocked.LastUpdate, Timezone)

	if err1 == nil && err2 == nil && err3 == nil {
		if triggerPackage.Data.Value == "RAIN" &&
			keyPosition.Value == "OFF" &&
			doorsLocked.Value == "TRUE" &&
			windowsOpen.Value == "TRUE" &&
			delta.Minutes() > 5 {
			slackAlert("Windows are down in the rain, eh?")
		}
	}
}
