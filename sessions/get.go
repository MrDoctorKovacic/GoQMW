package sessions

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/format/response"
	"github.com/rs/zerolog/log"
)

// HandleGetAll responds to an HTTP request for the entire session
func HandleGetAll(w http.ResponseWriter, r *http.Request) {
	requestingMin := r.URL.Query().Get("min") == "1"
	response := response.JSONResponse{OK: true}
	if requestingMin {
		response.Output = GetAllMin()
	} else {
		response.Output = GetAll()
	}
	response.Write(&w, r)
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
	response := response.JSONResponse{Output: sessionValue, OK: true}
	if err != nil {
		response.Output = err.Error()
		response.OK = false
	}
	response.Write(&w, r)
}

// Get returns the named session, if it exists. Nil otherwise
func Get(name string) (value string, err error) {
	session.Mutex.RLock()
	defer session.Mutex.RUnlock()
	sessionValue, ok := session.data[name]
	session.stats.Gets++

	if !ok {
		return "", fmt.Errorf("%s does not exist in Session", name)
	}
	return sessionValue.Value, nil
}

// GetData returns the named session, if it exists. Nil otherwise
func GetData(name string) (data Data, err error) {
	session.Mutex.RLock()
	defer session.Mutex.RUnlock()
	sessionValue, ok := session.data[name]
	session.stats.Gets++

	if !ok {
		return sessionValue, fmt.Errorf("%s does not exist in Session", name)
	}
	return sessionValue, nil
}

// GetLastModified returns the named session, if it exists. Nil otherwise
func GetLastModified(name string) (value string, err error) {
	session.Mutex.RLock()
	defer session.Mutex.RUnlock()
	sessionValue, ok := session.data[name]
	session.stats.Gets++

	if !ok {
		return "", fmt.Errorf("%s does not exist in Session", name)
	}
	return sessionValue.LastUpdate, nil
}

// GetBool returns the named session with a boolean value, on failure will return the default value
func GetBool(name string, defaultValue bool) bool {
	v, err := Get(name)
	if err != nil {
		return defaultValue
	}

	vb, err := strconv.ParseBool(v)
	if err != nil {
		return defaultValue
	}
	return vb
}

// GetString generalizes fetching session string with a default value
func GetString(name string, def string) string {
	v, err := Get(name)
	if err != nil {
		log.Trace().Msgf("%s could not be determined, defaulting to FALSE", name)
		v = def
	}
	return v
}
