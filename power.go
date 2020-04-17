package main

import (
	"fmt"
	"time"

	"github.com/qcasey/MDroid-Core/format"
	"github.com/qcasey/MDroid-Core/mserial"
	"github.com/qcasey/MDroid-Core/sessions"
	"github.com/qcasey/MDroid-Core/settings"
	"github.com/rs/zerolog/log"
)

// Define temporary holding struct for device values
type device struct {
	isOn       bool
	target     string
	settings   settingDef
	errors     errorType
	powerStats powerStats
}

type settingDef struct {
	component string
	name      string
}

type errorType struct {
	target error
	on     error
}

type powerStats struct {
	powerOnTime      time.Time
	lastTrigger      powerTrigger
	workingOnRequest bool
}

type powerTrigger struct {
	target string
	time   time.Time
}

// Read the target action based on current ACC Power value
var (
	_lock   = device{settings: settingDef{component: "MDROID", name: "AUTOLOCK"}}
	_angel  = device{settings: settingDef{component: "ANGEL_EYES", name: "POWER"}}
	_tablet = device{settings: settingDef{component: "TABLET", name: "POWER"}}
	_board  = device{settings: settingDef{component: "BOARD", name: "POWER"}}
)

func (ps *powerStats) startRequest() {
	ps.workingOnRequest = true
}

func (ps *powerStats) endRequest() {
	ps.workingOnRequest = false
}

// Evaluates if the doors should be locked
func evalAutoLock(keyIsIn string, accOn bool, wifiOn bool) {
	// Check if request is already being made
	if _lock.powerStats.workingOnRequest {
		return
	}

	// Very dumb lock to shoot down spam requests
	_lock.powerStats.startRequest()
	defer _lock.powerStats.endRequest()

	_lock.isOn, _lock.errors.on = sessions.GetBool("DOORS_LOCKED")
	_lock.target, _lock.errors.target = settings.Get(_lock.settings.component, _lock.settings.name)
	shouldTrigger := !_lock.isOn && !accOn && !wifiOn && keyIsIn == "FALSE"

	if _lock.errors.on != nil {
		// Don't log, likely just doesn't exist in session yet
		return
	}
	if _lock.errors.target != nil {
		log.Error().Msgf("Setting Error: %s", _lock.errors.target.Error())
		if _lock.settings.component != "" && _lock.settings.name != "" {
			log.Error().Msg("Setting read error for AUTOLOCK. Resetting to AUTO")
			settings.Set(_lock.settings.component, _lock.settings.name, "AUTO")
		}
		return
	}

	// Instead of power trigger, evaluate here. Lock once every so often
	if _lock.target == "AUTO" && shouldTrigger {
		lastLock, err := sessions.Get("DOORS_LOCKED")
		if err != nil {
			log.Error().Msg(_lock.errors.on.Error())
			return
		}
		lockToggleTime, err := time.Parse("", lastLock.LastUpdate)
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
func evalAutoSleep(keyIsIn string, accOn bool, wifiOn bool) {
	sleepEnabled, err := settings.Get("MDROID", "AUTO_SLEEP")

	if err != nil {
		log.Error().Msgf("Setting Error: %s", err.Error())
		log.Error().Msg("Setting read error for AUTO_SLEEP. Resetting to AUTO")
		settings.Set("MDROID", "AUTO_SLEEP", "ON")
		return
	}

	// If "OFF", auto sleep is not enabled. Exit
	if sleepEnabled != "ON" {
		return
	}

	// Don't fall asleep if the board was recently started
	if time.Since(sessions.GetStartTime()) < time.Minute*10 {
		return
	}

	// Sleep indefinitely, hand power control to the arduino
	if !accOn && wifiOn && keyIsIn == "FALSE" {
		sleepMDroid()
	}
}

// Evaluates if the angel eyes should be on, and then passes that struct along as generic power module
func evalAngelEyesPower(keyIsIn string) {
	_angel.isOn, _angel.errors.on = sessions.GetBool("ANGEL_EYES_POWER")
	_angel.target, _angel.errors.target = settings.Get(_angel.settings.component, _angel.settings.name)
	lightSensor := sessions.GetBoolDefault("LIGHT_SENSOR_ON", false)

	shouldTrigger := !lightSensor && keyIsIn != "FALSE"
	triggerReason := fmt.Sprintf("lightSensor: %t, keyIsIn: %s", lightSensor, keyIsIn)

	// Pass angel module to generic power trigger
	genericPowerTrigger(shouldTrigger, triggerReason, "Angel", &_angel)
}

// Evaluates if the video boards should be on, and then passes that struct along as generic power module
func evalVideoPower(keyIsIn string, accOn bool, wifiOn bool) {
	_board.isOn, _board.errors.on = sessions.GetBool("BOARD_POWER")
	_board.target, _board.errors.target = settings.Get(_board.settings.component, _board.settings.name)
	startedRecently := time.Since(sessions.GetStartTime()) < time.Minute*5

	shouldTrigger := (accOn && !wifiOn && !startedRecently) || ((wifiOn || startedRecently) && keyIsIn != "FALSE")
	triggerReason := fmt.Sprintf("accOn: %t, wifiOn: %t, keyIsIn: %s, startedRecently: %t", accOn, wifiOn, keyIsIn, startedRecently)

	// Pass angel module to generic power trigger
	genericPowerTrigger(shouldTrigger, triggerReason, "Board", &_board)
}

// Evaluates if the tablet should be on, and then passes that struct along as generic power module
func evalTabletPower(keyIsIn string, accOn bool, wifiOn bool) {
	_tablet.isOn, _tablet.errors.on = sessions.GetBool("TABLET_POWER")
	_tablet.target, _tablet.errors.target = settings.Get(_tablet.settings.component, _tablet.settings.name)
	startedRecently := time.Since(sessions.GetStartTime()) < time.Minute*5

	shouldTrigger := (accOn && !wifiOn && !startedRecently) || ((wifiOn || startedRecently) && keyIsIn != "FALSE")
	triggerReason := fmt.Sprintf("accOn: %t, wifiOn: %t, keyIsIn: %s, startedRecently: %t", accOn, wifiOn, keyIsIn, startedRecently)

	// Pass angel module to generic power trigger
	genericPowerTrigger(shouldTrigger, triggerReason, "Tablet", &_tablet)
}

// Error check against module's status fetches, then check if we're powering on or off
func genericPowerTrigger(shouldBeOn bool, reason string, name string, module *device) {
	// Check if request is already being made
	if module.powerStats.workingOnRequest {
		return
	}

	// Very dumb lock to shoot down spam requests
	module.powerStats.startRequest()
	defer module.powerStats.endRequest()

	// Handle error in fetches
	if module.errors.target != nil {
		log.Error().Msgf("Setting Error: %s", module.errors.target.Error())
		if module.settings.component != "" && module.settings.name != "" {
			log.Error().Msgf("Setting read error for %s. Resetting to AUTO", name)
			settings.Set(module.settings.component, module.settings.name, "AUTO")
		}
		return
	}
	if module.errors.on != nil {
		log.Debug().Msgf("Session Error: %s", module.errors.on.Error())
		return
	}

	// Add a limit to how many checks can occur
	if module.powerStats.lastTrigger.target != module.target && time.Since(module.powerStats.lastTrigger.time) < time.Second*3 {
		log.Info().Msgf("Ignoring target %s on module %s, since last check was under 3 seconds ago", name, module.target)
		return
	}

	var triggerType string
	// Evaluate power target with trigger and settings info
	if (module.target == "AUTO" && !module.isOn && shouldBeOn) || (module.target == "ON" && !module.isOn) {
		message := mserial.Message{Device: mserial.Writer, Text: fmt.Sprintf("powerOn%s", name)}
		triggerType = "on"
		mserial.Await(&message)
	} else if (module.target == "AUTO" && module.isOn && !shouldBeOn) || (module.target == "OFF" && module.isOn) {
		triggerType = "off"
		gracefulShutdown(name)
	} else {
		return
	}

	// Log and set next time threshold
	if module.target != "AUTO" {
		reason = fmt.Sprintf("target is %s", module.target)
	}
	log.Info().Msgf("Powering %s %s, because %s", triggerType, name, reason)

	module.powerStats.lastTrigger = powerTrigger{time: time.Now(), target: module.target}
}

// Some shutdowns are more complicated than others, ensure we shut down safely
func gracefulShutdown(name string) {
	serialCommand := fmt.Sprintf("powerOff%s", name)

	if name == "Board" || name == "Wireless" {
		err := sendServiceCommand(format.Name(name), "shutdown")
		if err != nil {
			log.Error().Msg(err.Error())
		}
		time.Sleep(time.Second * 10)
		err = mserial.AwaitText(serialCommand)
		if err != nil {
			log.Error().Msg(err.Error())
		}
	} else {
		err := mserial.AwaitText(serialCommand)
		if err != nil {
			log.Error().Msg(err.Error())
		}
	}
}
