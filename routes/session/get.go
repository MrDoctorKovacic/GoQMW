package session

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/internal/core"
	"github.com/qcasey/MDroid-Core/internal/core/sessions"
)

// HandleGet returns a specific session value
func HandleGet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	sessionValue := sessions.Data.Get(params["name"])
	response := core.JSONResponse{Output: sessionValue, OK: true}
	if !sessions.Data.IsSet(params["name"]) {
		response.Output = "Does not exist"
		response.OK = false
	}
	response.Write(&w, r)
}

// HandleGetAll responds to an HTTP request for the entire session
func HandleGetAll(w http.ResponseWriter, r *http.Request) {
	//requestingMin := r.URL.Query().Get("min") == "1"
	response := core.JSONResponse{OK: true}
	response.Output = sessions.Data.AllSettings()
	response.Write(&w, r)
}
