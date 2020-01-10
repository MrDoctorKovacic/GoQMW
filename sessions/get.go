package sessions

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/qcasey/MDroid-Core/format"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// HandleGetAll responds to an HTTP request for the entire session
func HandleGetAll(w http.ResponseWriter, r *http.Request) {
	requestingMin := r.URL.Query().Get("min") == "1"
	response := format.JSONResponse{OK: true}
	if requestingMin {
		response.Output = GetAllMin()
	} else {
		response.Output = GetAll()
	}
	format.WriteResponse(&w, r, response)
}

// GetAll returns the entire current session
func GetAll() map[string]Data {
	// Log if requested
	log.Debug().Msg("Responding to request for full session")

	newData := map[string]Data{}
	session.Mutex.RLock()
	defer session.Mutex.RUnlock()
	for index, element := range session.data {
		newData[index] = element
	}

	return newData
}

// GetAllMin returns the entire current session, minus unnecc values
func GetAllMin() map[string]string {
	// Log if requested
	log.Debug().Msg("Responding to request for minimal session")

	newData := map[string]string{}
	session.Mutex.RLock()
	defer session.Mutex.RUnlock()
	for index, element := range session.data {
		newData[index] = element.Value
	}

	return newData
}

// HandleGet returns a specific session value
func HandleGet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	sessionValue, err := Get(params["name"])
	response := format.JSONResponse{Output: sessionValue, OK: true}
	if err != nil {
		response.Output = err.Error()
		response.OK = false
	}
	format.WriteResponse(&w, r, response)
}

// Get returns the named session, if it exists. Nil otherwise
func Get(name string) (data Data, err error) {
	session.Mutex.RLock()
	defer session.Mutex.RUnlock()
	sessionValue, ok := session.data[name]
	session.stats.Gets++

	if !ok {
		return sessionValue, fmt.Errorf("%s does not exist in Session", name)
	}
	return sessionValue, nil
}

// GetBool returns the named session with a boolean value, if it exists. false otherwise
func GetBool(name string) (value bool, err error) {
	v, err := Get(name)
	if err != nil {
		return false, err
	}

	vb, err := strconv.ParseBool(v.Value)
	if err != nil {
		return false, err
	}
	return vb, nil
}

// GetStringDefault generalizes fetching session string
func GetStringDefault(name string, def string) string {
	v, err := Get(name)
	if err != nil {
		log.Debug().Msgf("%s could not be determined, defaulting to FALSE", name)
		v.Value = def
	}
	return v.Value
}

// GetBoolDefault generalizes fetching session bool
func GetBoolDefault(name string, def bool) bool {
	v, err := GetBool(name)
	if err != nil {
		log.Debug().Msgf("%s could not be determined, defaulting to false", name)
		v = def
	}
	return v
}
