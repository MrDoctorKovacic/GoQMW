package settings

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/internal/core"
	"github.com/qcasey/MDroid-Core/internal/core/settings"
	"github.com/rs/zerolog/log"
)

// HandleGetAll returns all current settings
func HandleGetAll(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msg("Responding to GET request with entire settings map.")
	resp := core.JSONResponse{Output: settings.Data.AllSettings(), Status: "success", OK: true}
	resp.Write(&w, r)
}

// HandleGet returns all the values of a specific setting
func HandleGet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	componentName := core.FormatName(params["key"])

	log.Debug().Msgf("Responding to GET request for setting component %s", componentName)

	resp := core.JSONResponse{Output: settings.Data.Get(params["key"]), OK: true}
	if !settings.Data.IsSet(params["key"]) {
		resp = core.JSONResponse{Output: "Setting not found.", OK: false}
	}

	resp.Write(&w, r)
}
