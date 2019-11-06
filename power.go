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
	_wireless = power{settingComp: "WIRELESS", settingName: "POWER"}
	_angel    = power{settingComp: "ANGEL_EYES", settingName: "POWER"}
	_tablet   = power{settingComp: "TABLET", settingName: "POWER"}
	_board    = power{settingComp: "BOARD", settingName: "POWER"}
)

// Evaluates if the angel eyes should be on, and then passes that struct along as generic power module
func evalAngelEyesPower(keyIsIn string) {
	_angel.on, _angel.errOn = sessions.GetBool("ANGEL_EYES_POWER")
	_angel.powerTarget, _angel.errTarget = settings.Get(_angel.settingComp, _angel.settingName)
	lightSensor := sessions.GetBoolDefault("LIGHT_SENSOR_ON", false)

	shouldTrigger := !lightSensor && keyIsIn != "FALSE"

	// Pass angel module to generic power trigger
	genericPowerTrigger(shouldTrigger, "Angel", &_angel)
}

// Evaluates if the video boards should be on, and then passes that struct along as generic power module
func evalVideoPower(keyIsIn string, accOn bool, wifiOn bool) {
	_board.on, _board.errOn = sessions.GetBool("BOARD_POWER")
	_board.powerTarget, _board.errTarget = settings.Get(_board.settingComp, _board.settingName)

	shouldTrigger := accOn && !wifiOn || wifiOn && keyIsIn != "FALSE"

	// Pass angel module to generic power trigger
	genericPowerTrigger(shouldTrigger, "Board", &_board)
}

// Evaluates if the tablet should be on, and then passes that struct along as generic power module
func evalTabletPower(keyIsIn string, accOn bool, wifiOn bool) {
	_tablet.on, _tablet.errOn = sessions.GetBool("TABLET_POWER")
	_tablet.powerTarget, _tablet.errTarget = settings.Get(_tablet.settingComp, _tablet.settingName)

	shouldTrigger := accOn && !wifiOn || wifiOn && keyIsIn != "FALSE"

	// Pass angel module to generic power trigger
	genericPowerTrigger(shouldTrigger, "Tablet", &_tablet)
}

// Evaluates if the wireless boards should be on, and then passes that struct along as generic power module
func evalWirelessPower(keyIsIn string, accOn bool, wifiOn bool) {
	_wireless.on, _wireless.errOn = sessions.GetBool("WIRELESS_POWER")
	_wireless.powerTarget, _wireless.errTarget = settings.Get(_wireless.settingComp, _wireless.settingName)

	// Wireless is most likely supposed to be on, only one case where it should not be
	shouldTrigger := true
	if wifiOn && keyIsIn == "FALSE" {
		shouldTrigger = false
	}

	// Pass wireless module to generic power trigger
	genericPowerTrigger(shouldTrigger, "Wireless", &_wireless)
}

// Error check against module's status fetches, then check if we're powering on or off
func genericPowerTrigger(shouldBeOn bool, name string, module *power) {
	// Handle error in fetches
	if module.errTarget != nil {
		log.Error().Msg(fmt.Sprintf("Setting Error: %s", module.errTarget.Error()))
		if module.settingComp != "" && module.settingName != "" {
			log.Error().Msg(fmt.Sprintf("Setting read error for %s. Resetting to AUTO", name))
			settings.Set(module.settingComp, module.settingName, "AUTO")
		}
		return
	}
	if module.errOn != nil {
		log.Debug().Msg(fmt.Sprintf("Session Error: %s", module.errOn.Error()))
		return
	}

	// Evaluate power target with trigger and settings info
	if (module.powerTarget == "AUTO" && !module.on && shouldBeOn) || (module.powerTarget == "ON" && !module.on) {
		log.Info().Msg(fmt.Sprintf("Powering on %s, because target is %s", name, module.powerTarget))
		mserial.Push(mserial.Writer, fmt.Sprintf("powerOn%s", name))
	} else if (module.powerTarget == "AUTO" && module.on && !shouldBeOn) || (module.powerTarget == "OFF" && module.on) {
		log.Info().Msg(fmt.Sprintf("Powering off %s, because target is %s", name, module.powerTarget))
		gracefulShutdown(name)
	}
}

// Some shutdowns are more complicated than others, ensure we shut down safely
func gracefulShutdown(name string) {
	serialCommand := fmt.Sprintf("powerOff%s", name)

	if name == "Board" || name == "Wireless" {
		sendServiceCommand(format.Name(name), "shutdown")
		time.Sleep(time.Second * 10)
		mserial.Push(mserial.Writer, serialCommand)
	} else {
		mserial.Push(mserial.Writer, serialCommand)
	}
}
