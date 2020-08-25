package main

import (
	"fmt"
	"time"

	"github.com/qcasey/MDroid-Core/bluetooth"
	"github.com/qcasey/MDroid-Core/mserial"
	"github.com/qcasey/MDroid-Core/sessions"
	"github.com/qcasey/MDroid-Core/settings"
	"github.com/rs/zerolog/log"
)

func isOnAuxPower() bool {
	return sessions.Data.GetBool("acc_power.value")
}

func isHome() bool {
	return sessions.Data.GetBool("acc_power.value")
}

func addCustomHooks() {
	// When ACC power state is changed
	sessions.HL.RegisterHook("ACC_POWER", time.Second*5, evalAngelEyesPower, evalBluetoothDeviceState, evalLowPowerMode, evalAutoLock, evalAutoSleep)

	settings.HL.RegisterHook("AUTO_SLEEP", -1, evalAutoSleep)
	settings.HL.RegisterHook("AUTO_LOCK", -1, evalAutoLock)
	settings.HL.RegisterHook("ANGEL_EYES", -1, evalAngelEyesPower)
	sessions.HL.RegisterHook("LIGHT_SENSOR_REASON", -1, lightSensorReason)
	sessions.HL.RegisterHook("LIGHT_SENSOR_ON", -1, evalAngelEyesPower)
	sessions.HL.RegisterHook("SEAT_MEMORY_1", -1, func() { sendServiceCommand("MDROID", "restart") })
}

// Alert me when it's raining and windows are down
func lightSensorReason() {
	keyPosition := sessions.Data.GetString("key_position.value")
	windowsOpen := sessions.Data.GetString("windows_open.value")
	doorsLocked := sessions.Data.GetString("doors_locked.value")

	if sessions.Data.GetString("light_sensor_reason.value") == "RAIN" &&
		keyPosition == "OFF" &&
		doorsLocked == "TRUE" &&
		windowsOpen == "TRUE" {
		sessions.SlackAlert("Windows are down in the rain, eh?")
	}
}

func evalBluetoothDeviceState() {
	if sessions.Data.GetString("connected_bluetooth_device.value") == "" {
		return
	}

	// Play / pause bluetooth media on key in/out
	if isOnAuxPower() {
		bluetooth.Play()
	} else {
		bluetooth.Pause()
	}
}

// Evaluates if the doors should be locked
func evalAutoLock() {
	if !sessions.Data.IsSet("doors_locked") {
		// Likely just doesn't exist in session yet
		return
	}

	// Instead of power trigger, evaluate here. Lock once every so often
	if settings.Get("mdroid.autolock", "AUTO") == "AUTO" &&
		sessions.Data.GetString("doors_locked.value") == "FALSE" &&
		!isOnAuxPower() &&
		!isHome() {

		lockToggleTime, err := time.Parse("", sessions.Data.GetString("doors_locked.write_date"))
		if err != nil {
			log.Error().Msg(err.Error())
			return
		}

		// For debugging
		log.Info().Msg(lockToggleTime.String())

		// handle case where car is UNLOCKED recently, i.e. getting back in before putting key in
		if time.Since(lockToggleTime) < time.Minute*5 {
			return
		}

		//_lock.lastCheck = triggerType{time: time.Now(), target: _lock.target}
		err = mserial.AwaitText("toggleDoorLocks")
		if err != nil {
			log.Error().Msg(err.Error())
		}
	}
}

// Evaluates if the board should be put to sleep
func evalAutoSleep() {
	// If "OFF", auto sleep is not enabled.
	if settings.Data.GetString("mdroid.auto_sleep") != "ON" {
		return
	}

	// Don't fall asleep if the board was recently started
	if time.Since(sessions.GetStartTime()) < time.Minute*10 {
		return
	}

	// Sleep indefinitely, hand power control to the arduino
	if !isOnAuxPower() && isHome() {
		sleepMDroid()
	}
}

// Evaluates if the angel eyes should be on, and then passes that struct along as generic power module
func evalAngelEyesPower() {
	hasPower := isOnAuxPower()
	lightSensor := sessions.Data.GetString("light_sensor_on.value") == "FALSE"

	shouldBeOn := lightSensor && hasPower
	reason := fmt.Sprintf("lightSensor: %t, hasPower: %t", lightSensor, hasPower)

	// Pass angel module to generic power trigger
	genericTrigger("ANGEL_EYES", shouldBeOn, reason)
}

// Evaluates if the cameras and tablet should be on, and then passes that struct along as generic power module
func evalLowPowerMode() {
	accOn := isOnAuxPower()
	isHome := isHome()
	startedRecently := time.Since(sessions.GetStartTime()) < time.Minute*5

	shouldBeOn := (accOn && !isHome && !startedRecently) || (isHome || startedRecently)
	reason := fmt.Sprintf("accOn: %t, isHome: %t, startedRecently: %t", accOn, isHome, startedRecently)

	// Pass angel module to generic power trigger
	genericTrigger("USB_HUB", shouldBeOn, reason)
}

func genericTriggerWithCustomFunctions(componentName string, shouldBeOn bool, reason string, onFunction func(), offFunction func()) {
	moduleIsOn := sessions.Data.GetBool(fmt.Sprintf("%s.value", componentName))
	moduleSetting := settings.Data.GetString(fmt.Sprintf("%s.power", componentName))

	var triggerType string
	// Evaluate power target with trigger and settings info
	if (moduleSetting == "AUTO" && !moduleIsOn && shouldBeOn) || (moduleSetting == "ON" && !moduleIsOn) {
		triggerType = "on"
		onFunction()
	} else if (moduleSetting == "AUTO" && moduleIsOn && !shouldBeOn) || (moduleSetting == "OFF" && moduleIsOn) {
		triggerType = "off"
		offFunction()
	} else {
		return
	}

	// Log and set next time threshold
	if moduleSetting != "AUTO" {
		reason = fmt.Sprintf("target is %s", moduleSetting)
	}
	log.Info().Msgf("Powering %s %s, because %s", triggerType, componentName, reason)
}

// Error check against module's status fetches, then check if we're powering on or off
func genericTrigger(componentName string, shouldBeOn bool, reason string) {
	on := func() {
		err := mserial.AwaitText(fmt.Sprintf("powerOn:%s", componentName))
		if err != nil {
			log.Error().Msg(err.Error())
		}
	}
	off := func() {
		err := mserial.AwaitText(fmt.Sprintf("powerOff:%s", componentName))
		if err != nil {
			log.Error().Msg(err.Error())
		}
	}
	genericTriggerWithCustomFunctions(componentName, shouldBeOn, reason, on, off)
}
