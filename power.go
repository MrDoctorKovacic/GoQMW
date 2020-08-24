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

func hasAuxPower() bool {
	return sessions.Data.GetBool("acc_power.value")
}

func evalBluetoothDeviceState() {
	// Play / pause bluetooth media on key in/out
	if sessions.Data.GetString("connected_bluetooth_device.value") != "" {
		if hasAuxPower() {
			bluetooth.Play()
		} else {
			bluetooth.Pause()
		}
	}
}

// Evaluates if the doors should be locked
func evalAutoLock() {
	accOn := hasAuxPower()
	isHome := sessions.Data.GetBool("ble_connected.value")

	if !sessions.Data.IsSet("doors_locked") {
		// Don't log, likely just doesn't exist in session yet
		return
	}

	target := settings.Get("mdroid.autolock", "auto")
	shouldBeOn := sessions.Data.GetString("doors_locked.value") == "FALSE" && !accOn && !isHome

	// Instead of power trigger, evaluate here. Lock once every so often
	if target == "AUTO" && shouldBeOn {
		lockToggleTime, err := time.Parse("", sessions.Data.GetString("doors_locked.write_date"))
		if err != nil {
			log.Error().Msg(err.Error())
			return
		}

		// For debugging
		log.Info().Msg(lockToggleTime.String())
		//lockedInLast15Mins := time.Since(_lock.lastCheck.time) < time.Minute*15
		unlockedInLast5Minutes := time.Since(lockToggleTime) < time.Minute*5 // handle case where car is UNLOCKED recently, i.e. getting back in. Before putting key in

		if unlockedInLast5Minutes {
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
	accOn := hasAuxPower()
	isHome := sessions.Data.GetBool("ble_connected.value")
	sleepEnabled := settings.Data.GetString("mdroid.auto_sleep")

	// If "OFF", auto sleep is not enabled. Exit
	if sleepEnabled != "ON" {
		return
	}

	// Don't fall asleep if the board was recently started
	if time.Since(sessions.GetStartTime()) < time.Minute*10 {
		return
	}

	// Sleep indefinitely, hand power control to the arduino
	if !accOn && isHome {
		sleepMDroid()
	}
}

// Evaluates if the angel eyes should be on, and then passes that struct along as generic power module
func evalAngelEyesPower() {
	hasPower := hasAuxPower()
	lightSensor := sessions.Data.GetString("light_sensor_on.value") == "FALSE"

	shouldBeOn := lightSensor && hasPower
	triggerReason := fmt.Sprintf("lightSensor: %t, hasPower: %t", lightSensor, hasPower)

	// Pass angel module to generic power trigger
	powerTrigger(shouldBeOn, triggerReason, "ANGEL_EYES")
}

// Evaluates if the cameras and tablet should be on, and then passes that struct along as generic power module
func evalLowPowerMode() {
	accOn := sessions.Data.GetBool("acc_power.value")
	isHome := sessions.Data.GetBool("ble_connected.value")
	startedRecently := time.Since(sessions.GetStartTime()) < time.Minute*5

	shouldBeOn := (accOn && !isHome && !startedRecently) || (isHome || startedRecently)
	triggerReason := fmt.Sprintf("accOn: %t, isHome: %t, startedRecently: %t", accOn, isHome, startedRecently)

	// Pass angel module to generic power trigger
	powerTrigger(shouldBeOn, triggerReason, "USB_HUB")
}

// Error check against module's status fetches, then check if we're powering on or off
func powerTrigger(shouldBeOn bool, reason string, componentName string) {
	moduleIsOn := sessions.Data.GetBool(fmt.Sprintf("%s.value", componentName))
	moduleSetting := settings.Data.GetString(fmt.Sprintf("%s.power", componentName))

	var triggerType string
	// Evaluate power target with trigger and settings info
	if (moduleSetting == "AUTO" && !moduleIsOn && shouldBeOn) || (moduleSetting == "ON" && !moduleIsOn) {
		triggerType = "on"
		mserial.AwaitText(fmt.Sprintf("powerOn:%s", componentName))
	} else if (moduleSetting == "AUTO" && moduleIsOn && !shouldBeOn) || (moduleSetting == "OFF" && moduleIsOn) {
		triggerType = "off"
		mserial.AwaitText(fmt.Sprintf("powerOff:%s", componentName))
	} else {
		return
	}

	// Log and set next time threshold
	if moduleSetting != "AUTO" {
		reason = fmt.Sprintf("target is %s", moduleSetting)
	}
	log.Info().Msgf("Powering %s %s, because %s", triggerType, componentName, reason)
}
