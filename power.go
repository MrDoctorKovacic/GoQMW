package main

import (
	"fmt"

	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/rs/zerolog/log"
)

// Evaluates if the angel eyes should be on, and then passes that struct along as generic power module
func evalAngelEyesPower(keyIsIn string) {
	angel := angelDef
	angel.on, angel.errOn = sessions.GetBool("ANGEL_EYES_POWER")
	angel.powerTarget, angel.errTarget = settings.Get(angel.settingComp, angel.settingName)
	lightSensor := sessions.GetBoolDefault("LIGHT_SENSOR_ON", false)

	shouldTrigger := !lightSensor && keyIsIn != "FALSE"

	// Pass angel module to generic power trigger
	genericPowerTrigger(shouldTrigger, "Angel", angel)
}

// Evaluates if the video boards should be on, and then passes that struct along as generic power module
func evalVideoPower(keyIsIn string) {
	board := boardDef
	board.on, board.errOn = sessions.GetBool("BOARD_POWER")
	board.powerTarget, board.errTarget = settings.Get(board.settingComp, board.settingName)

	// Pass angel module to generic power trigger
	genericPowerTrigger(keyIsIn != "FALSE", "Board", board)
}

// Evaluates if the wireless boards should be on, and then passes that struct along as generic power module
func evalWirelessPower(accOn bool, wifiOn bool) {
	wireless := wirelessDef
	wireless.on, wireless.errOn = sessions.GetBool("WIRELESS_POWER")
	wireless.powerTarget, wireless.errTarget = settings.Get(wireless.settingComp, wireless.settingName)

	// Wireless is most likely supposed to be on, only one case where it should not be
	shouldTrigger := true
	if !accOn && wifiOn {
		shouldTrigger = false
	}

	// Pass wireless module to generic power trigger
	genericPowerTrigger(shouldTrigger, "Wireless", wireless)
}

// Evaluates if the sound board should be on, and then passes that struct along as generic power module
func evalSoundPower(accOn bool, wifiOn bool) {
	sound := soundDef
	sound.on, sound.errOn = sessions.GetBool("SOUND_POWER")
	sound.powerTarget, sound.errTarget = settings.Get(sound.settingComp, sound.settingName)

	keyIsIn := sessions.GetStringDefault("KEY_STATE", "FALSE")
	shouldTrigger := accOn && !wifiOn || wifiOn && keyIsIn != "FALSE"

	// Pass sound module to generic power trigger
	genericPowerTrigger(shouldTrigger, "Sound", sound)
}

// Error check against module's status fetches, then check if we're powering on or off
func genericPowerTrigger(shouldBeOn bool, name string, module power) {
	if module.errOn == nil && module.errTarget == nil {
		if (module.powerTarget == "AUTO" && !module.on && shouldBeOn) || (module.powerTarget == "ON" && !module.on) {
			log.Info().Msg(fmt.Sprintf("Powering on %s, because target is %s", name, module.powerTarget))
			mserial.Push(mserial.Writer, fmt.Sprintf("powerOn%s", name))
		} else if (module.powerTarget == "AUTO" && module.on && !shouldBeOn) || (module.powerTarget == "OFF" && module.on) {
			log.Info().Msg(fmt.Sprintf("Powering off %s, because target is %s", name, module.powerTarget))
			gracefulShutdown(name)
		}
	} else if module.errTarget != nil {
		log.Error().Msg(fmt.Sprintf("Setting Error: %s", module.errTarget.Error()))
		if module.settingComp != "" && module.settingName != "" {
			log.Error().Msg(fmt.Sprintf("Setting read error for %s. Resetting to AUTO", name))
			settings.Set(module.settingComp, module.settingName, "AUTO")
		}
	} else if module.errOn != nil {
		log.Debug().Msg(fmt.Sprintf("Session Error: %s", module.errOn.Error()))
	}
}
