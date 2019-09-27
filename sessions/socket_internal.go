package sessions

import (
	"net/http"

	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/gorilla/websocket"
)

// Session WebSocket upgrader
var upgrader = websocket.Upgrader{} // use default options

// GetSessionSocket returns the entire current session as a webstream
func (session *Session) GetSessionSocket(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true } // return true for now, although this should range over accepted origins

	// Log if requested
	if settings.Config.VerboseOutput {
		status.Log(logging.OK(), "Responding to request for session websocket")
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		status.Log(logging.Error(), "Error upgrading webstream: "+err.Error())
		return
	}
	defer c.Close()
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			status.Log(logging.Error(), "Error reading from webstream: "+err.Error())
			break
		}

		// Very verbose
		//status.Log(logging.OK(), "Received: "+string(message))

		// Pass through lock first
		writeSession := session.GetSession()

		err = c.WriteJSON(writeSession)

		if err != nil {
			status.Log(logging.Error(), "Error writing to webstream: "+err.Error())
			break
		}
	}
}
