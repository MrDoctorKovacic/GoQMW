package settings

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/internal/core"
	"github.com/rs/zerolog/log"
)

// GetAll returns all current settings
func GetAll(c *core.Core) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debug().Msg("Responding to GET request with entire settings map.")
		resp := core.JSONResponse{Output: c.Settings.AllSettings(), Status: "success", OK: true}
		resp.Write(&w, r)
	}
}

// Get returns all the values of a specific setting
func Get(c *core.Core) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		componentName := core.FormatName(params["key"])

		log.Debug().Msgf("Responding to GET request for setting component %s", componentName)

		resp := core.JSONResponse{Output: c.Settings.Get(params["key"]), OK: true}
		if !c.Settings.IsSet(params["key"]) {
			resp = core.JSONResponse{Output: "Setting not found.", OK: false}
		}

		resp.Write(&w, r)
	}
}
