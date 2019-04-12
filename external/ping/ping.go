package ping

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/MrDoctorKovacic/GoQMW/influx"
	"github.com/gorilla/mux"
)

// DB variables
var DB influx.Influx
var remote string

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
			log.Println("[Ping] Logging " + params["device"] + " to database")

			// Insert into database
			err := DB.Write(fmt.Sprintf("ping,device=%s ip=\"%s\"", params["device"], params["ip"]))

			if err != nil {
				log.Println(err.Error())
			} else {
				log.Println("Logged " + params["device"] + " to database")
			}

		} else {
			//
			// FWD request to server since we have internet
			//
			log.Println("[Ping] Forwarding " + params["device"] + " to server")
			pingResp, err := http.Get(remote + "?name=" + params["device"] + "&local_ip=" + params["ip"])
			if err != nil {
				defer pingResp.Body.Close()
				log.Println("Error when forwarding ping:")
				log.Println(err)
			}
		}
	}

	// Devices will not act on response anyway, anything but 200 is a waste
	json.NewEncoder(w).Encode("OK")
}

// DumpPings is ideally run when reconnected to internet. Will dump local pings to remote server
func DumpPings(w http.ResponseWriter, r *http.Request) {

}
