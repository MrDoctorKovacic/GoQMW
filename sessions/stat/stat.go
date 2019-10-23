// Package stat implements session values regarding system stats, including CPU, RAM, Disk usage and temps
package stat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/MrDoctorKovacic/MDroid-Core/format"
	"github.com/MrDoctorKovacic/MDroid-Core/influx"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// stat holds various data points we expect to receive
type stat struct {
	UsedRAM     float32 `json:"usedRAM,omitempty"`
	UsedCPU     float32 `json:"usedCPU,omitempty"`
	UsedDisk    float32 `json:"usedDisk,omitempty"`
	UsedNetwork float32 `json:"usedNetwork,omitempty"`
	TempCPU     float32 `json:"tempCPU,omitempty"`
}

// status will control logging and reporting of status / warnings / errors
var (
	stats     map[string]stat
	statsLock sync.Mutex
)

func init() {
	stats = make(map[string]stat, 0)
}

// HandleGet returns the latest stat
func HandleGet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	statResponse, ok := get(params["name"])
	format.WriteResponse(&w, r, format.JSONResponse{Output: statResponse, OK: ok})
}

func get(name string) (stat, bool) {
	statsLock.Lock()
	defer statsLock.Unlock()
	statResponse, ok := stats[format.Name(name)]
	return statResponse, ok
}

// HandleSet posts a new stat
func HandleSet(w http.ResponseWriter, r *http.Request) {
	var newdata stat
	if err := json.NewDecoder(r.Body).Decode(&newdata); err != nil {
		log.Error().Msg(err.Error())
		return
	}

	params := mux.Vars(r)
	formattedName := format.Name(params["name"])
	statsLock.Lock()
	stats[formattedName] = newdata
	statsLock.Unlock()

	// Insert into database
	if influx.DB != nil {

		fields := map[string]interface{}{
			"cpu":     newdata.UsedCPU,
			"ram":     newdata.UsedRAM,
			"disk":    newdata.UsedDisk,
			"network": newdata.UsedNetwork,
			"temp":    newdata.TempCPU,
		}

		err := influx.DB.Insert("stats", map[string]interface{}{"name": formattedName}, fields)
		if err != nil && influx.DB.Started {
			log.Error().Msg(fmt.Sprintf("Error writing string stats to influx DB: %s", err.Error()))
			return
		}
		log.Debug().Msg(fmt.Sprintf("Logged stats to database"))
	}
	format.WriteResponse(&w, r, format.JSONResponse{Output: formattedName, OK: true})
}
