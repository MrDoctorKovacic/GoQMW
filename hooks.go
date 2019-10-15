package main

import "github.com/MrDoctorKovacic/MDroid-Core/settings"

func setupHooks() {
	settings.RegisterHook("ANGEL_EYES", angelEyesSettings)
}

func angelEyesSettings(settingName string, settingValue string) {
	switch settingName {
	case "POWER":
	}
}
