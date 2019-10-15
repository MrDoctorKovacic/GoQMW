package main

import (
	"fmt"
	"math"
	"strconv"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/gps"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions"

	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
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
	wirelessDef = power{settingComp: "LTE", settingName: "POWER"}
	wifiDef     = power{settingComp: "", settingName: ""}
	angelDef    = power{settingComp: "ANGEL_EYES", settingName: "POWER"}
	tabletDef   = power{settingComp: "TABLET", settingName: "POWER"}
	boardDef    = power{settingComp: "BOARD", settingName: "POWER"}
	hookStatus  logging.ProgramStatus
)

func init() {
	hookStatus = logging.NewStatus("Hooks")
}

func setupHooks() {
	settings.RegisterHook("ANGEL_EYES", angelEyesSettings)
	sessions.RegisterHookSlice(&[]string{"MAIN_VOLTAGE_RAW", "AUX_VOLTAGE_RAW"}, voltage)
	sessions.RegisterHook("AUX_CURRENT_RAW", auxCurrent)
	sessions.RegisterHook("ACC_POWER", accPower)
	sessions.RegisterHook("KEY_STATE", keyState)
	sessions.RegisterHook("WIRELESS_POWER", lteOn)
	sessions.RegisterHook("LIGHT_SENSOR_REASON", lightSensorReason)
	sessions.RegisterHook("LIGHT_SENSOR_ON", lightSensorOn)
	sessions.RegisterHookSlice(&[]string{"SEAT_MEMORY_1", "SEAT_MEMORY_2", "SEAT_MEMORY_3"}, voltage)
}

//
// From here on out are the hook functions.
// We're taking actions based on the values or a combination of values
// from the session/settings post values.
//

func angelEyesSettings(settingName string, settingValue string) {
	// Determine state of angel eyes
	evalAngelEyes()
}

func keyState(hook *sessions.SessionPackage) {
	// Determine state of angel eyes
	evalAngelEyes()
}

func lightSensorOn(hook *sessions.SessionPackage) {
	// Determine state of angel eyes
	evalAngelEyes()
}

func evalAngelEyes() {
	angel := angelDef
	angel.on, angel.errOn = sessions.GetBool("ANGEL_EYES_POWER")
	angel.powerTarget, angel.errTarget = settings.Get(angel.settingComp, angel.settingName)

	keyIsIn, err := sessions.Get("KEY_STATE")
	if err != nil {
		keyIsIn.Value = "FALSE"
	}

	lightSensor, err := sessions.GetBool("LIGHT_SENSOR_ON")
	if err != nil {
		lightSensor = false
	}

	shouldTrigger := !lightSensor && keyIsIn.Value != "FALSE"

	// Pass angel module to generic power trigger
	genericPowerTrigger(shouldTrigger, "Angel", angel)
}

// Convert main raw voltage into an actual number
func voltage(hook *sessions.SessionPackage) {
	voltageFloat, err := strconv.ParseFloat(hook.Data.Value, 64)
	if err != nil {
		hookStatus.Log(logging.Error(), fmt.Sprintf("Failed to convert string %s to float", hook.Data.Value))
		return
	}

	sessions.SetValue(hook.Name[0:len(hook.Name)-4], fmt.Sprintf("%.3f", (voltageFloat/1024)*24.4))
}

// Modifiers to the incoming Current sensor value
func auxCurrent(hook *sessions.SessionPackage) {
	currentFloat, err := strconv.ParseFloat(hook.Data.Value, 64)

	if err != nil {
		hookStatus.Log(logging.Error(), fmt.Sprintf("Failed to convert string %s to float", hook.Data.Value))
		return
	}

	realCurrent := math.Abs(1000 * ((((currentFloat * 3.3) / 4095.0) - 1.5) / 185))
	sessions.SetValue("AUX_CURRENT", fmt.Sprintf("%.3f", realCurrent))
}

// Trigger for booting boards/tablets
func accPower(hook *sessions.SessionPackage) {
	// Read the target action based on current ACC Power value
	var accOn bool
	wireless := wirelessDef
	wifi := wifiDef
	tablet := tabletDef
	board := boardDef

	// Check incoming ACC power value is valid
	switch hook.Data.Value {
	case "TRUE":
		accOn = true
	case "FALSE":
		accOn = false
	default:
		hookStatus.Log(logging.Error(), fmt.Sprintf("ACC Power Trigger unexpected value: %s", hook.Data.Value))
		return
	}

	// Verbose, but pull all the necessary configuration data
	wireless.on, wireless.errOn = sessions.GetBool("WIRELESS_POWER")
	wireless.powerTarget, wireless.errTarget = settings.Get(wireless.settingComp, wireless.settingName)
	wifi.on, wifi.errOn = sessions.GetBool("WIFI_CONNECTED")
	tablet.on, tablet.errOn = sessions.GetBool("TABLET_POWER")
	tablet.powerTarget, tablet.errTarget = settings.Get(tablet.settingComp, tablet.settingName)
	board.on, board.errOn = sessions.GetBool("BOARD_POWER")
	board.powerTarget, board.errTarget = settings.Get(board.settingComp, board.settingName)

	// Trigger wireless, based on wifi status
	if wifi.errOn == nil && wireless.errOn == nil && wireless.errTarget == nil {
		if wireless.powerTarget == "AUTO" && !wifi.on && !wireless.on {
			wireless.powerTarget = "ON"
		}
	}

	// Handle more generic modules
	modules := map[string]power{"Board": board, "Tablet": tablet, "Wireless": wireless}

	for name, module := range modules {
		go genericPowerTrigger(accOn, name, module)
	}
}

// Error check against module's status fetches, then check if we're powering on or off
func genericPowerTrigger(accOn bool, name string, module power) {
	if module.errOn == nil && module.errTarget == nil {
		if (module.powerTarget == "AUTO" && !module.on && accOn) || (module.powerTarget == "ON" && !module.on) {
			hookStatus.Log(logging.OK(), fmt.Sprintf("Powering on %s, because target is %s", name, module.powerTarget))
			mserial.Push(settings.Config.SerialControlDevice, fmt.Sprintf("powerOn%s", name))
		} else if (module.powerTarget == "AUTO" && module.on && !accOn) || (module.powerTarget == "OFF" && module.on) {
			hookStatus.Log(logging.OK(), fmt.Sprintf("Powering off %s, because target is %s", name, module.powerTarget))
			gracefulShutdown(name)
		}
	} else if module.errTarget != nil {
		hookStatus.Log(logging.Error(), fmt.Sprintf("Setting Error: %s", module.errTarget.Error()))
		if module.settingComp != "" && module.settingName != "" {
			hookStatus.Log(logging.Error(), fmt.Sprintf("Setting read error for %s. Resetting to AUTO", name))
			settings.Set(module.settingComp, module.settingName, "AUTO")
		}
	} else if module.errOn != nil {
		hookStatus.Log(logging.Error(), fmt.Sprintf("Session Error: %s", module.errOn.Error()))
	}
}

func lteOn(hook *sessions.SessionPackage) {
	lteOn, err := sessions.Get("WIRELESS_POWER")
	if err != nil {
		hookStatus.Log(logging.Error(), err.Error())
		return
	}

	if hook.Data.Value == "FALSE" && lteOn.Value == "TRUE" {
		// When board is turned off but doesn't have time to reflect LTE status
		sessions.SetValue("LTE_ON", "FALSE")
	}
}

// Alert me when it's raining and windows are down
func lightSensorReason(hook *sessions.SessionPackage) {
	keyPosition, _ := sessions.Get("KEY_POSITION")
	doorsLocked, _ := sessions.Get("DOORS_LOCKED")
	windowsOpen, _ := sessions.Get("WINDOWS_OPEN")
	delta, err := formatting.CompareTimeToNow(doorsLocked.LastUpdate, gps.GetTimezone())

	if err != nil {
		if hook.Data.Value == "RAIN" &&
			keyPosition.Value == "OFF" &&
			doorsLocked.Value == "TRUE" &&
			windowsOpen.Value == "TRUE" &&
			delta.Minutes() > 5 {
			logging.SlackAlert(settings.Config.SlackURL, "Windows are down in the rain, eh?")
		}
	}
}

// Restart different machines when seat memory buttons are pressed
func seatMemory(hook *sessions.SessionPackage) {
	switch hook.Name {
	case "SEAT_MEMORY_1":
		mserial.CommandNetworkMachine("BOARD", "restart")
	case "SEAT_MEMORY_2":
		mserial.CommandNetworkMachine("LTE", "restart")
	case "SEAT_MEMORY_3":
		mserial.CommandNetworkMachine("MDROID", "restart")
	}
}
