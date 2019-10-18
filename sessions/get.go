package sessions

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// HandleGetAll responds to an HTTP request for the entire session
func HandleGetAll(w http.ResponseWriter, r *http.Request) {
	requestingMin := r.URL.Query().Get("min") == "1"
	response := formatting.JSONResponse{OK: true}
	if requestingMin {
		response.Output = GetAllMin()
	} else {
		response.Output = GetAll()
	}
	formatting.WriteResponse(&w, response)
}

// GetAll returns the entire current session
func GetAll() map[string]Value {
	// Log if requested
	log.Debug().Msg("Responding to request for full session")

	newData := map[string]Value{}
	session.Mutex.Lock()
	defer session.Mutex.Unlock()
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
	session.Mutex.Lock()
	defer session.Mutex.Unlock()
	for index, element := range session.data {
		newData[index] = element.Value
	}

	return newData
}

// HandleGet returns a specific session value
func HandleGet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	sessionValue, err := Get(params["name"])
	response := formatting.JSONResponse{Output: sessionValue, OK: true}
	if err != nil {
		response.Output = err.Error()
		response.OK = false
	}
	formatting.WriteResponse(&w, response)
}

// Get returns the named session, if it exists. Nil otherwise
func Get(name string) (value Value, err error) {

	// Log if requested
	log.Debug().Msg(fmt.Sprintf("Responding to request for session value %s", name))

	session.Mutex.Lock()
	defer session.Mutex.Unlock()
	sessionValue, ok := session.data[name]

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
		log.Debug().Msg(fmt.Sprintf("%s could not be determined, defaulting to FALSE", name))
		v.Value = def
	}
	return v.Value
}

// GetBoolDefault generalizes fetching session bool
func GetBoolDefault(name string, def bool) bool {
	v, err := GetBool(name)
	if err != nil {
		log.Debug().Msg(fmt.Sprintf("%s could not be determined, defaulting to false", name))
		v = def
	}
	return v
}
