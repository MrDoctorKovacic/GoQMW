package sessions

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// Session WebSocket upgrader
var upgrader = websocket.Upgrader{} // use default options

// GetSessionSocket returns the entire current session as a webstream
func GetSessionSocket(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true } // return true for now, although this should range over accepted origins

	// Log if requested
	log.Debug().Msg("Responding to request for session websocket")

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Msg("Error upgrading webstream: " + err.Error())
		return
	}
	defer c.Close()
	for {
		if _, _, err := c.ReadMessage(); err != nil {
			log.Error().Msg("Error reading from webstream: " + err.Error())
			break
		}

		// Pass through lock first
		writeSession := GetAll()
		if err = c.WriteJSON(writeSession); err != nil {
			log.Error().Msg("Error writing to webstream: " + err.Error())
			break
		}
	}
}
