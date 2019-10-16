package logging

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// Ping will fwd to remote server if connected to internet, otherwise will log locally
func Ping(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	// Ensure we have a server (and a DB) to connect to
	if RemotePingAddress != "" {
		onlineResp, err := http.Get("1.1.1.1")

		if err != nil {
			//
			// Log locally
			//
			defer onlineResp.Body.Close()
			status.Log(OK(), fmt.Sprintf("Logging %s to database", params["device"]))

			// Insert into database
			//err := DB.Write(fmt.Sprintf("ping,device=%s ip=\"%s\"", params["device"], params["ip"]))

			if err != nil {
				status.Log(Error(), fmt.Sprintf("Error when logging %s to database: %s", params["device"], err.Error()))
			} else {
				status.Log(OK(), fmt.Sprintf("Logged %s to database", params["device"]))
			}

		} else {
			// Forward request to server since we have internet
			status.Log(OK(), fmt.Sprintf("Forwarding %s to server", params["device"]))
			pingResp, err := http.Get(fmt.Sprintf("%s?name=%s&local_ip=%s", RemotePingAddress, params["device"], params["ip"]))
			if err != nil {
				defer pingResp.Body.Close()
				status.Log(Error(), fmt.Sprintf("Error when forwarding ping: %s", err.Error()))
			}
		}

		json.NewEncoder(w).Encode("OK")
	}

	// Devices will not act on response anyway, anything but 200 is a waste
	json.NewEncoder(w).Encode("OK")
}

// DumpPings is ideally run when reconnected to internet.
// Will dump local pings to remote server
func DumpPings(w http.ResponseWriter, r *http.Request) {

}
