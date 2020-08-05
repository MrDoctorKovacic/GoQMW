package main

import (
	bluetooth "github.com/qcasey/MDroid-Bluetooth"
	"github.com/qcasey/MDroid-Core/format"
	"github.com/qcasey/MDroid-Core/sessions"
	"github.com/qcasey/MDroid-Core/sessions/gps"
	"github.com/rs/zerolog/log"

	"github.com/qcasey/MDroid-Core/settings"
)

func setupHooks() {
	settings.RegisterHook("AUTO_SLEEP", autoSleepSettings)
	settings.RegisterHook("AUTO_LOCK", autoLockSettings)
	settings.RegisterHook("ANGEL_EYES", angelEyesSettings)
	sessions.RegisterHook("ACC_POWER", accPower)
	sessions.RegisterHook("KEY_STATE", keyState)
	sessions.RegisterHook("LIGHT_SENSOR_REASON", lightSensorReason)
	sessions.RegisterHook("LIGHT_SENSOR_ON", lightSensorOn)
	sessions.RegisterHook("SEAT_MEMORY_1", seatMemory)
	log.Info().Msg("Enabled session hooks")
}

//
// From here on out are the hook functions.
// We're taking actions based on the values or a combination of values
// from the session/settings.
//

// When angel eyes setting is changed
func angelEyesSettings(key string, value interface{}) {
	// Determine state of angel eyes
	go evalAngelEyesPower()
}

// When auto lock setting is changed
func autoLockSettings(key string, value interface{}) {
	// Trigger state of auto lock
	go evalAutoLock()
}

// When auto Sleep setting is changed
func autoSleepSettings(key string, value interface{}) {
	// Trigger state of auto sleep
	go evalAutoSleep()
}

// When ACC power state is changed
func accPower(hook *sessions.Data) {
	// Trigger low power and auto sleep
	go evalLowPowerMode()
	go evalAutoLock()
	go evalAutoSleep()
}

// When key state is changed
func keyState(hook *sessions.Data) {
	// Play / pause bluetooth media on key in/out
	if hook.Value != "FALSE" {
		go bluetooth.Play()
	} else {
		go bluetooth.Pause()
	}

	// Determine state of angel eyes, and main board
	go evalAngelEyesPower()
	go evalLowPowerMode()
	go evalAutoLock()
}

// When light sensor is changed in session
func lightSensorOn(hook *sessions.Data) {
	// Determine state of angel eyes
	go evalAngelEyesPower()
}

// Alert me when it's raining and windows are down
func lightSensorReason(hook *sessions.Data) {
	keyPosition := sessions.GetString("KEY_POSITION", "OFF")
	windowsOpen := sessions.GetBool("WINDOWS_OPEN", false)
	doorsLocked, err := sessions.GetData("DOORS_LOCKED")

	delta, err := format.CompareTimeToNow(doorsLocked.LastUpdate, gps.GetTimezone())
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	if hook.Value == "RAIN" &&
		keyPosition == "OFF" &&
		doorsLocked.Value == "TRUE" &&
		windowsOpen &&
		delta.Minutes() > 5 {
		sessions.SlackAlert("Windows are down in the rain, eh?")
	}
}

// Restart different machines when seat memory buttons are pressed
func seatMemory(hook *sessions.Data) {
	if hook.Name == "SEAT_MEMORY_1" {
		sendServiceCommand("MDROID", "restart")
	}
}
