package bluetooth

import (
	"regexp"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/settings"
)

// Bluetooth is the modular implementation of Bluetooth controls
type Bluetooth struct {
	BluetoothAddress string
	tmuxStarted      bool
	replySerialRegex *regexp.Regexp
	findStringRegex  *regexp.Regexp
	cleanRegex       *regexp.Regexp
}

// Mod exports our Bluetooth
var Mod Bluetooth

func init() {
	Mod.replySerialRegex = regexp.MustCompile(`(.*reply_serial=2\n\s*variant\s*)array`)
	Mod.findStringRegex = regexp.MustCompile(`string\s"(.*)"|uint32\s(\d)+`)
	Mod.cleanRegex = regexp.MustCompile(`(string|uint32|\")+`)
}

// Setup bluetooth with address
func (Bluetooth) Setup() {
	SetAddress(settings.Data.GetString("mdroid.BLUETOOTH_ADDRESS"))
	go startAutoRefresh()
}

// SetRoutes handles module routing
func (Bluetooth) SetRoutes(router *mux.Router) {
	//
	// Bluetooth routes
	//
	router.HandleFunc("/bluetooth", GetDeviceInfo).Methods("GET")
	router.HandleFunc("/bluetooth/getDeviceInfo", GetDeviceInfo).Methods("GET")
	router.HandleFunc("/bluetooth/getMediaInfo", GetMediaInfo).Methods("GET")
	router.HandleFunc("/bluetooth/connect", HandleConnect).Methods("GET")
	router.HandleFunc("/bluetooth/disconnect", HandleDisconnect).Methods("GET")
	router.HandleFunc("/bluetooth/prev", Prev).Methods("GET")
	router.HandleFunc("/bluetooth/next", Next).Methods("GET")
	router.HandleFunc("/bluetooth/pause", HandlePause).Methods("GET")
	router.HandleFunc("/bluetooth/play", HandlePlay).Methods("GET")
	router.HandleFunc("/bluetooth/refresh", ForceRefresh).Methods("GET")
}
