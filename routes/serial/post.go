package serial

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/internal/core"
	"github.com/qcasey/MDroid-Core/pkg/mserial"
)

// WriteSerialHandler handles messages sent through the server
func WriteSerialHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	if params["command"] != "" {
		mserial.AwaitText(params["command"])
	}
	core.WriteNewResponse(&w, r, core.JSONResponse{Output: "OK", OK: true})
}
