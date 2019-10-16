package sessions

import (
	"net/http"

	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/gorilla/websocket"
)

// Session WebSocket upgrader
var upgrader = websocket.Upgrader{} // use default options

// GetSessionSocket returns the entire current session as a webstream
func GetSessionSocket(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true } // return true for now, although this should range over accepted origins

	// Log if requested
	status.Log(logging.Debug(), "Responding to request for session websocket")

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		status.Log(logging.Error(), "Error upgrading webstream: "+err.Error())
		return
	}
	defer c.Close()
	for {
		if _, _, err := c.ReadMessage(); err != nil {
			status.Log(logging.Error(), "Error reading from webstream: "+err.Error())
			break
		}

		// Pass through lock first
		writeSession := GetAll()
		if err = c.WriteJSON(writeSession); err != nil {
			status.Log(logging.Error(), "Error writing to webstream: "+err.Error())
			break
		}
	}
}
