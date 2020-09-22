package session

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/internal/core"
)

// GetAll responds to an HTTP request for the entire session
func GetAll(c *core.Core) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//requestingMin := r.URL.Query().Get("min") == "1"
		response := core.JSONResponse{OK: true}
		response.Output = c.Session.AllSettings()
		response.Write(&w, r)
	}
}

// Get returns a specific session value
func Get(c *core.Core) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		params := mux.Vars(r)

		sessionValue := c.Session.Get(params["name"])
		response := core.JSONResponse{Output: sessionValue, OK: true}
		if !c.Session.IsSet(params["name"]) {
			response.Output = "Does not exist"
			response.OK = false
		}
		response.Write(&w, r)
	}
}
