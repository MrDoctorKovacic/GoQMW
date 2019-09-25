package sessions

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/gorilla/websocket"
)

// Session WebSocket upgrader
var upgrader = websocket.Upgrader{} // use default options

// GetSessionSocket returns the entire current session as a webstream
func (session *Session) GetSessionSocket(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true } // return true for now, although this should range over accepted origins

	// Log if requested
	if session.Config.VerboseOutput {
		SessionStatus.Log(logging.OK(), "Responding to request for session websocket")
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		SessionStatus.Log(logging.Error(), "Error upgrading webstream: "+err.Error())
		return
	}
	defer c.Close()
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			SessionStatus.Log(logging.Error(), "Error reading from webstream: "+err.Error())
			break
		}

		// Very verbose
		//SessionStatus.Log(logging.OK(), "Received: "+string(message))

		// Pass through lock first
		writeSession := session.GetSession()

		err = c.WriteJSON(writeSession)

		if err != nil {
			SessionStatus.Log(logging.Error(), "Error writing to webstream: "+err.Error())
			break
		}
	}
}

// CheckServer will continiously ping a central server for waiting packets,
// and will open a websocket as a client if so
func (session *Session) CheckServer(host string, token string) {
	lteEnabled, err := session.GetSessionValue("LTE_ON")
	if err != nil {
		SessionStatus.Log(logging.Error(), "Error getting LTE status.")
		return
	}
	for lteEnabled.Value == "TRUE" {
		lteEnabled, err = session.GetSessionValue("LTE_ON")
		if err != nil {
			SessionStatus.Log(logging.Error(), "Error getting LTE status.")
			return
		}

		resp, err := http.Get(fmt.Sprintf("%s/ping", host))
		if err != nil {
			// handle error
			SessionStatus.Log(logging.Error(), fmt.Sprintf("Error when pinging the central server.\n%s", err.Error()))
			return
		}
		resp.Body.Close()
		if resp.StatusCode == 200 {
			requestServerSocket(fmt.Sprintf("%s/%s", host, token))
		}

		time.Sleep(time.Second * 10)
	}
}

func requestServerSocket(host string) {

}
