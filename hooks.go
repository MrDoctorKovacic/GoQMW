package main

import (
	"fmt"
	"math"
	"strconv"

	"github.com/MrDoctorKovacic/MDroid-Core/format"
	"github.com/MrDoctorKovacic/MDroid-Core/gps"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions"
	"github.com/rs/zerolog/log"

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
	_soundDef    = power{settingComp: "SOUND", settingName: "POWER"}
	_wirelessDef = power{settingComp: "WIRELESS", settingName: "POWER"}
	_angelDef    = power{settingComp: "ANGEL_EYES", settingName: "POWER"}
	_tabletDef   = power{settingComp: "TABLET", settingName: "POWER"}
	_boardDef    = power{settingComp: "BOARD", settingName: "POWER"}
)

func setupHooks() {
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

// When wireless setting is changed
func wirelessSettings(settingName string, settingValue string) {
	accOn := sessions.GetBoolDefault("ACC_POWER", false)
	wifiOn := sessions.GetBoolDefault("WIFI_CONNECTED", false)

	// Determine state of wireless
	evalWirelessPower(accOn, wifiOn)
}

// When key state is changed in session
func keyState(hook *sessions.SessionPackage) {
	accOn := sessions.GetBoolDefault("ACC_POWER", false)
	wifiOn := sessions.GetBoolDefault("WIFI_CONNECTED", false)

	// Determine state of angel eyes
	evalAngelEyesPower(hook.Data.Value)

	// Determine state of the video boards
	evalVideoPower(hook.Data.Value, accOn, wifiOn)

	// Determine state of the sound board
	evalSoundPower(hook.Data.Value, accOn, wifiOn)
}

// When light sensor is changed in session
func lightSensorOn(hook *sessions.SessionPackage) {
	// Determine state of angel eyes
	evalAngelEyesPower(sessions.GetStringDefault("KEY_STATE", "FALSE"))
}

// Convert main raw voltage into an actual number
func voltage(hook *sessions.SessionPackage) {
	voltageFloat, err := strconv.ParseFloat(hook.Data.Value, 64)
	if err != nil {
		log.Error().Msg(fmt.Sprintf("Failed to convert string %s to float", hook.Data.Value))
		return
	}

	sessions.SetValue(hook.Name[0:len(hook.Name)-4], fmt.Sprintf("%.3f", (voltageFloat/1024)*24.4))
}

// Modifiers to the incoming Current sensor value
func auxCurrent(hook *sessions.SessionPackage) {
	currentFloat, err := strconv.ParseFloat(hook.Data.Value, 64)
	if err != nil {
		log.Error().Msg(fmt.Sprintf("Failed to convert string %s to float", hook.Data.Value))
		return
	}

	realCurrent := math.Abs(1000 * ((((currentFloat * 3.3) / 4095.0) - 1.5) / 185))
	sessions.SetValue("AUX_CURRENT", fmt.Sprintf("%.3f", realCurrent))
}

// Trigger for booting boards/tablets
func accPower(hook *sessions.SessionPackage) {
	// Read the target action based on current ACC Power value
	var accOn bool

	// Check incoming ACC power value is valid
	switch hook.Data.Value {
	case "TRUE":
		accOn = true
	case "FALSE":
		accOn = false
	default:
		log.Error().Msg(fmt.Sprintf("ACC Power Trigger unexpected value: %s", hook.Data.Value))
		return
	}

	// Pull the necessary configuration data
	wifiOn := sessions.GetBoolDefault("WIFI_CONNECTED", false)
	keyIsIn := sessions.GetStringDefault("KEY_STATE", "FALSE")

	// Trigger wireless, based on ACC and wifi status
	go evalWirelessPower(accOn, wifiOn)

	// Trigger video, based on ACC and wifi status
	go evalVideoPower(keyIsIn, accOn, wifiOn)

	// Trigger sound, based on ACC and wifi status
	go evalSoundPower(keyIsIn, accOn, wifiOn)

	// Trigger sound, based on ACC and wifi status
	go evalTabletPower(keyIsIn, accOn, wifiOn)
}

// When wireless is turned off, we can infer that LTE is also off
func wirelessPower(hook *sessions.SessionPackage) {
	if hook.Data.Value == "FALSE" {
		// When board is turned off but doesn't have time to reflect LTE status
		sessions.SetValue("LTE_ON", "FALSE")
	}
}

// Alert me when it's raining and windows are down
func lightSensorReason(hook *sessions.SessionPackage) {
	keyPosition, _ := sessions.Get("KEY_POSITION")
	doorsLocked, _ := sessions.Get("DOORS_LOCKED")
	windowsOpen, _ := sessions.Get("WINDOWS_OPEN")
	delta, err := format.CompareTimeToNow(doorsLocked.LastUpdate, gps.GetTimezone())

	if err != nil {
		if hook.Data.Value == "RAIN" &&
			keyPosition.Value == "OFF" &&
			doorsLocked.Value == "TRUE" &&
			windowsOpen.Value == "TRUE" &&
			delta.Minutes() > 5 {
			sessions.SlackAlert(settings.SlackURL, "Windows are down in the rain, eh?")
		}
	}
}

// Restart different machines when seat memory buttons are pressed
func seatMemory(hook *sessions.SessionPackage) {
	switch hook.Name {
	case "SEAT_MEMORY_1":
		sendServiceCommand("BOARD", "restart")
	case "SEAT_MEMORY_2":
		sendServiceCommand("WIRELESS", "restart")
	case "SEAT_MEMORY_3":
		sendServiceCommand("MDROID", "restart")
	}
}
