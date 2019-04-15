package ping

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MrDoctorKovacic/GoQMW/external/status"
	"github.com/MrDoctorKovacic/GoQMW/influx"
	"github.com/gorilla/mux"
)

// DB variables
var DB influx.Influx
var remote string

// PingStatus will control logging and reporting of status / warnings / errors
var PingStatus = status.NewStatus("Ping")

// Setup Database for future queries, setup fwding address
func Setup(inf influx.Influx, remoteAddress string) {
	DB = inf
	remote = remoteAddress
}

// Ping will fwd to remote server if connected to internet, otherwise will log locally
func Ping(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	// Ensure we have a server (and a DB) to connect to
	if remote != "" {
		onlineResp, err := http.Get("1.1.1.1")

		if err != nil {
			//
			// Log locally
			//
			defer onlineResp.Body.Close()
			PingStatus.Log(status.OK(), "Logging "+params["device"]+" to database")

			// Insert into database
			err := DB.Write(fmt.Sprintf("ping,device=%s ip=\"%s\"", params["device"], params["ip"]))

			if err != nil {
				PingStatus.Log(status.Error(), "Error when logging "+params["device"]+" to database: "+err.Error())
			} else {
				PingStatus.Log(status.OK(), "Logged "+params["device"]+" to database")
			}

		} else {
			//
			// FWD request to server since we have internet
			//
			PingStatus.Log(status.OK(), "Forwarding "+params["device"]+" to server")
			pingResp, err := http.Get(remote + "?name=" + params["device"] + "&local_ip=" + params["ip"])
			if err != nil {
				defer pingResp.Body.Close()
				PingStatus.Log(status.Error(), "Error when forwarding ping: "+err.Error())
			}
		}
	}

	// Devices will not act on response anyway, anything but 200 is a waste
	json.NewEncoder(w).Encode("OK")
}

// DumpPings is ideally run when reconnected to internet. Will dump local pings to remote server
func DumpPings(w http.ResponseWriter, r *http.Request) {

}
