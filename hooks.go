package main

import (
	"fmt"
	"strconv"

	"github.com/MrDoctorKovacic/MDroid-Core/format"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions/gps"
	"github.com/rs/zerolog/log"

	"github.com/MrDoctorKovacic/MDroid-Core/settings"
)

func setupHooks() {
	settings.RegisterHook("AUTO_SLEEP", autoSleepSettings)
	settings.RegisterHook("AUTO_LOCK", autoLockSettings)
	settings.RegisterHook("ANGEL_EYES", angelEyesSettings)
	settings.RegisterHook("WIRELESS", wirelessSettings)
	sessions.RegisterHookSlice(&[]string{"MAIN_VOLTAGE_RAW", "AUX_VOLTAGE_RAW"}, voltage)
	sessions.RegisterHook("AUX_CURRENT_RAW", auxCurrent)
	sessions.RegisterHook("ACC_POWER", accPower)
	sessions.RegisterHook("KEY_STATE", keyState)
	sessions.RegisterHook("WIRELESS_POWER", wirelessPower)
	sessions.RegisterHook("LIGHT_SENSOR_REASON", lightSensorReason)
	sessions.RegisterHook("LIGHT_SENSOR_ON", lightSensorOn)
	sessions.RegisterHookSlice(&[]string{"SEAT_MEMORY_1", "SEAT_MEMORY_2", "SEAT_MEMORY_3"}, voltage)
}

//
// From here on out are the hook functions.
// We're taking actions based on the values or a combination of values
// from the session/settings post values.
//

// When angel eyes setting is changed
func angelEyesSettings(settingName string, settingValue string) {
	// Determine state of angel eyes
	evalAngelEyesPower(sessions.GetStringDefault("KEY_STATE", "FALSE"))
}

// When auto lock setting is changed
func autoLockSettings(settingName string, settingValue string) {
	accOn := sessions.GetBoolDefault("ACC_POWER", false)
	wifiOn := sessions.GetBoolDefault("WIFI_CONNECTED", true)

	// Determine state of auto lock
	evalAutoLock(sessions.GetStringDefault("KEY_STATE", "FALSE"), accOn, wifiOn)
}

// When auto Sleep setting is changed
func autoSleepSettings(settingName string, settingValue string) {
	accOn := sessions.GetBoolDefault("ACC_POWER", false)
	wifiOn := sessions.GetBoolDefault("WIFI_CONNECTED", true)

	// Determine state of auto Sleep
	evalAutoSleep(sessions.GetStringDefault("KEY_STATE", "FALSE"), accOn, wifiOn)
}

// When wireless setting is changed
func wirelessSettings(settingName string, settingValue string) {
	accOn := sessions.GetBoolDefault("ACC_POWER", false)
	wifiOn := sessions.GetBoolDefault("WIFI_CONNECTED", true)

	// Determine state of wireless
	evalWirelessPower(sessions.GetStringDefault("KEY_STATE", "FALSE"), accOn, wifiOn)
}

// When key state is changed in session
func keyState(hook *sessions.Data) {
	accOn := sessions.GetBoolDefault("ACC_POWER", false)
	wifiOn := sessions.GetBoolDefault("WIFI_CONNECTED", true)

	// Determine state of wireless, angel eyes, and main board
	evalWirelessPower(hook.Value, accOn, wifiOn)
	evalAngelEyesPower(hook.Value)
	evalVideoPower(hook.Value, accOn, wifiOn)
	evalAutoLock(hook.Value, accOn, wifiOn)
}

// When light sensor is changed in session
func lightSensorOn(hook *sessions.Data) {
	// Determine state of angel eyes
	evalAngelEyesPower(sessions.GetStringDefault("KEY_STATE", "FALSE"))
}

// Convert main raw voltage into an actual number
func voltage(hook *sessions.Data) {
	voltageFloat, err := strconv.ParseFloat(hook.Value, 64)
	if err != nil {
		log.Error().Msgf("Failed to convert string %s to float", hook.Value)
		return
	}

	//sessions.SetValue(hook.Name[0:len(hook.Name)-4], fmt.Sprintf("%.3f", 0.15+(voltageFloat/1024)*24.4))
	sessions.SetValue(hook.Name[0:len(hook.Name)-4], fmt.Sprintf("%.3f", (voltageFloat/1024)*22))
}

// Modifiers to the incoming Current sensor value
func auxCurrent(hook *sessions.Data) {
	currentFloat, err := strconv.ParseFloat(hook.Value, 32)
	if err != nil {
		log.Error().Msgf("Failed to convert string %s to float", hook.Value)
		return
	}

	modifier := .06
	if currentFloat < .3 {
		modifier = .09
	} else if currentFloat > 1.5 {
		modifier = .3
	}

	realCurrent := currentFloat + modifier
	sessions.SetValue("AUX_CURRENT", fmt.Sprintf("%.3f", realCurrent))
}

// Trigger for booting boards/tablets
func accPower(hook *sessions.Data) {
	var accOn bool

	// Check incoming ACC power value is valid
	switch hook.Value {
	case "TRUE":
		accOn = true
	case "FALSE":
		accOn = false
	default:
		log.Error().Msgf("ACC Power Trigger unexpected value: %s", hook.Value)
		return
	}

	// Pull the necessary configuration data
	wifiOn := sessions.GetBoolDefault("WIFI_CONNECTED", true)
	keyIsIn := sessions.GetStringDefault("KEY_STATE", "FALSE")

	// Trigger wireless, video, and tablet based on ACC and wifi status
	go evalWirelessPower(keyIsIn, accOn, wifiOn)
	go evalVideoPower(keyIsIn, accOn, wifiOn)
	go evalTabletPower(keyIsIn, accOn, wifiOn)
	go evalAutoLock(keyIsIn, accOn, wifiOn)
}

// When wireless is turned off, we can infer that LTE is also off
func wirelessPower(hook *sessions.Data) {
	// Pull the necessary configuration data
	lteOn := sessions.GetBoolDefault("LTE_ON", false)

	// When board is turned off but doesn't have time to reflect LTE status
	if hook.Value == "FALSE" && lteOn {
		// Indicate LTE is off
		sessions.SetValue("LTE_ON", "FALSE")

		// Reset network (temporary fix for DNS issues on LTE -> WIFI transition)
		resetNetwork()
	}
}

// Alert me when it's raining and windows are down
func lightSensorReason(hook *sessions.Data) {
	keyPosition, _ := sessions.Get("KEY_POSITION")
	doorsLocked, _ := sessions.Get("DOORS_LOCKED")
	windowsOpen, _ := sessions.Get("WINDOWS_OPEN")
	delta, err := format.CompareTimeToNow(doorsLocked.LastUpdate, gps.GetTimezone())

	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	if hook.Value == "RAIN" &&
		keyPosition.Value == "OFF" &&
		doorsLocked.Value == "TRUE" &&
		windowsOpen.Value == "TRUE" &&
		delta.Minutes() > 5 {
		sessions.SlackAlert("Windows are down in the rain, eh?")
	}
}

// Restart different machines when seat memory buttons are pressed
func seatMemory(hook *sessions.Data) {
	switch hook.Name {
	case "SEAT_MEMORY_1":
		sendServiceCommand("BOARD", "restart")
	case "SEAT_MEMORY_2":
		sendServiceCommand("WIRELESS", "restart")
	case "SEAT_MEMORY_3":
		sendServiceCommand("MDROID", "restart")
	}
}
