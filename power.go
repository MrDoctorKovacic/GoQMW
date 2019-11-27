package main

import (
	"fmt"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/format"
	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
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
	_lock     = device{settings: settingDef{component: "MDROID", name: "AUTOLOCK"}}
	_wireless = device{settings: settingDef{component: "WIRELESS", name: "POWER"}}
	_angel    = device{settings: settingDef{component: "ANGEL_EYES", name: "POWER"}}
	_tablet   = device{settings: settingDef{component: "TABLET", name: "POWER"}}
	_board    = device{settings: settingDef{component: "BOARD", name: "POWER"}}
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
		mserial.AwaitText("toggleDoorLocks")
	}
}

// Evaluates if the board should be put to sleep
func evalAutoSleep(keyIsIn string, accOn bool, wifiOn bool) {
	sleepEnabled, err := settings.Get("MDROID", "SLEEP")

	if err != nil {
		log.Error().Msgf("Setting Error: %s", err.Error())
		log.Error().Msg("Setting read error for SLEEP. Resetting to AUTO")
		settings.Set("MDROID", "SLEEP", "AUTO")
		return
	}

	if sleepEnabled != "AUTO" {
		return
	}

	// Instead of power trigger, evaluate here. Sleep every so often
	now := time.Now().Local()
	var (
		isTimeToSleep = false
		msToSleep     time.Duration
	)
	if now.Hour() >= 20 {
		isTimeToSleep = true
		msToSleep = (time.Duration(30-now.Hour()) * time.Hour)
	} else if now.Hour() <= 5 {
		isTimeToSleep = true
		msToSleep = (time.Duration(6-now.Hour()) * time.Hour)
	}
	msToSleep = msToSleep + time.Duration(60-now.Minute())*time.Minute
	msToSleep = msToSleep + time.Duration(60-now.Second())*time.Second
	shouldTrigger := !accOn && wifiOn && keyIsIn == "FALSE" && isTimeToSleep

	if shouldTrigger {
		log.Info().Msgf("Going to sleep now, for %f minutes", msToSleep.Minutes())
		//mserial.Push(mserial.Writer, fmt.Sprintf("putToSleep%d", msToSleep.Milliseconds()))
		// shutdown now
		//sendServiceCommand("MDROID", "shutdown")
	}
}

// Evaluates if the angel eyes should be on, and then passes that struct along as generic power module
func evalAngelEyesPower(keyIsIn string) {
	_angel.isOn, _angel.errors.on = sessions.GetBool("ANGEL_EYES_POWER")
	_angel.target, _angel.errors.target = settings.Get(_angel.settings.component, _angel.settings.name)
	lightSensor := sessions.GetBoolDefault("LIGHT_SENSOR_ON", false)

	shouldTrigger := !lightSensor && keyIsIn != "FALSE"

	// Pass angel module to generic power trigger
	genericPowerTrigger(shouldTrigger, "Angel", &_angel)
}

// Evaluates if the video boards should be on, and then passes that struct along as generic power module
func evalVideoPower(keyIsIn string, accOn bool, wifiOn bool) {
	_board.isOn, _board.errors.on = sessions.GetBool("BOARD_POWER")
	_board.target, _board.errors.target = settings.Get(_board.settings.component, _board.settings.name)

	shouldTrigger := accOn && !wifiOn || wifiOn && keyIsIn != "FALSE"

	// Pass angel module to generic power trigger
	genericPowerTrigger(shouldTrigger, "Board", &_board)
}

// Evaluates if the tablet should be on, and then passes that struct along as generic power module
func evalTabletPower(keyIsIn string, accOn bool, wifiOn bool) {
	_tablet.isOn, _tablet.errors.on = sessions.GetBool("TABLET_POWER")
	_tablet.target, _tablet.errors.target = settings.Get(_tablet.settings.component, _tablet.settings.name)

	shouldTrigger := accOn && !wifiOn || wifiOn && keyIsIn != "FALSE"

	// Pass angel module to generic power trigger
	genericPowerTrigger(shouldTrigger, "Tablet", &_tablet)
}

// Evaluates if the wireless boards should be on, and then passes that struct along as generic power module
func evalWirelessPower(keyIsIn string, accOn bool, wifiOn bool) {
	_wireless.isOn, _wireless.errors.on = sessions.GetBool("WIRELESS_POWER")
	_wireless.target, _wireless.errors.target = settings.Get(_wireless.settings.component, _wireless.settings.name)

	// Wireless is most likely supposed to be on, only one case where it should not be
	shouldTrigger := true
	if wifiOn && keyIsIn == "FALSE" {
		shouldTrigger = false
	}

	// Pass wireless module to generic power trigger
	genericPowerTrigger(shouldTrigger, "Wireless", &_wireless)
}

// Error check against module's status fetches, then check if we're powering on or off
func genericPowerTrigger(shouldBeOn bool, name string, module *device) {
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

	// Evaluate power target with trigger and settings info
	if (module.target == "AUTO" && !module.isOn && shouldBeOn) || (module.target == "ON" && !module.isOn) {
		message := mserial.Message{Device: mserial.Writer, Text: fmt.Sprintf("powerOn%s", name)}
		mserial.Await(&message)
	} else if (module.target == "AUTO" && module.isOn && !shouldBeOn) || (module.target == "OFF" && module.isOn) {
		gracefulShutdown(name)
	} else {
		return
	}

	// Log and set next time threshold
	log.Info().Msgf("Powering off %s, because target is %s", name, module.target)
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
		mserial.AwaitText(serialCommand)
	} else {
		mserial.AwaitText(serialCommand)
	}
}
