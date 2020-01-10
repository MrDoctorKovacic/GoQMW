// Package stat implements session values regarding system stats, including CPU, RAM, Disk usage and temps
package stat

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/qcasey/MDroid-Core/db"
	"github.com/qcasey/MDroid-Core/format"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// stat holds various data points we expect to receive
type stat struct {
	Name        string  `json:"name,omitempty"`
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

// HandleGetAll returns all the latest stats
func HandleGetAll(w http.ResponseWriter, r *http.Request) {
	statsLock.Lock()
	defer statsLock.Unlock()
	format.WriteResponse(&w, r, format.JSONResponse{Output: stats, OK: true})
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

func getAll() map[string]stat {
	newData := map[string]stat{}
	statsLock.Lock()
	defer statsLock.Unlock()
	for index, element := range stats {
		newData[index] = element
	}

	return newData
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
	newdata.Name = formattedName
	statsLock.Lock()
	stats[formattedName] = newdata
	statsLock.Unlock()

	// Insert into database
	if db.DB != nil {

		fields := map[string]interface{}{
			"cpu":     newdata.UsedCPU,
			"ram":     newdata.UsedRAM,
			"disk":    newdata.UsedDisk,
			"network": newdata.UsedNetwork,
			"temp":    newdata.TempCPU,
		}

		err := db.DB.Insert("stats", map[string]interface{}{"name": formattedName}, fields)
		if err != nil && db.DB.Started {
			log.Error().Msgf("Error writing string stats to influx DB: %s", err.Error())
			return
		}
		log.Debug().Msgf("Logged stats to database")
	}
	format.WriteResponse(&w, r, format.JSONResponse{Output: formattedName, OK: true})
}
