package ping

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// DB MySQL variables
var DB *sql.DB
var remote string

// Setup MySQL for future queries, setup fwding address
func Setup(database *sql.DB, remoteAddress string) {
	DB = database
	remote = remoteAddress
}

// Ping will fwd to remote server if connected to internet, otherwise will log locally
func Ping(w http.ResponseWriter, r *http.Request) {
	// Ensure we have a server (and a DB) to connect to
	if remote != "" {
		params := mux.Vars(r)
		_, err := http.Get("1.1.1.1")

		if err != nil {
			//
			// Log locally
			//
			_, err := DB.Exec("INSERT INTO log_ping (timestamp, device, ip) values (?, ?, ?)", time.Now().Format("2006-01-02 15:04:05.999"), params["device"], params["ip"])

			if err != nil {
				log.Println("Error executing SQL insert:")
				log.Println(err.Error())
			} else {
				log.Println("Logged " + params["device"] + " to sql db")
			}

		} else {
			//
			// FWD request to server since we have internet
			//
			_, err := http.Get(remote + "?name=" + params["device"] + "&local_ip=" + params["ip"])
			if err != nil {
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
