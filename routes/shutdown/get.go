package shutdown

import (
	"net/http"

	"github.com/qcasey/MDroid-Core/internal/core"
)

// HandleShutdown the current machine
func HandleShutdown(w http.ResponseWriter, r *http.Request) {
	/*params := mux.Vars(r)
	machine, ok := params["machine"]

		if !ok {
			core.WriteNewResponse(&w, r, core.JSONResponse{Output: "Machine name required", OK: false})
			return
		}*/

	core.WriteNewResponse(&w, r, core.JSONResponse{Output: "OK", OK: true})
	/*err := sendServiceCommand(machine, "shutdown")
	if err != nil {
		log.Error().Msg(err.Error())

			go func() { mserial.PushText(fmt.Sprintf("putToSleep%d", -1)) }()
	//sendServiceCommand("MDROID", "shutdown")
	}*/
}
