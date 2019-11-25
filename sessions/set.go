package sessions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/format"
	"github.com/MrDoctorKovacic/MDroid-Core/influx"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions/gps"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// HandleSet updates or posts a new session value to the common session
func HandleSet(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)

	// Default to NOT OK response
	response := format.JSONResponse{OK: false}

	if err != nil {
		log.Error().Msgf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	// Put body back
	r.Body.Close() //  must close
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	if len(body) == 0 {
		response.Output = "Error: Empty body"
		format.WriteResponse(&w, r, response)
		return
	}

	params := mux.Vars(r)
	var newdata Data

	if err = json.NewDecoder(r.Body).Decode(&newdata); err != nil {
		log.Error().Msgf("Error decoding incoming JSON:\n%s", err.Error())
		response.Output = err.Error()
		format.WriteResponse(&w, r, response)
		return
	}

	// Call the setter
	newdata.Name = params["name"]
	if err = Set(newdata); err != nil {
		response.Output = err.Error()
		format.WriteResponse(&w, r, response)
		return
	}

	// Craft OK response
	response.OK = true
	response.Output = newdata

	format.WriteResponse(&w, r, response)
}

// SetValue prepares a Value structure before passing it to the setter
func SetValue(name string, value string) Data {
	newPackage := Data{Name: name, Value: value, Quiet: true}
	Set(newPackage)
	return newPackage
}

// Set does the actual setting of Session Values
func Set(newPackage Data) error {
	// Ensure name is valid
	if !format.IsValidName(newPackage.Name) {
		return fmt.Errorf("%s is not a valid name. Possibly a failed serial transmission?", newPackage.Name)
	}

	// Set last updated time to now
	newPackage.LastUpdate = time.Now().In(gps.GetTimezone()).Format("2006-01-02 15:04:05.999")

	// Correct name
	newPackage.Name = format.Name(newPackage.Name)

	// Trim off whitespace
	newPackage.Value = strings.TrimSpace(newPackage.Value)

	// Log if requested
	log.Debug().Msgf("Responding to request for session key %s = %s", newPackage.Name, newPackage.Value)

	// Add / update value in global session after locking access to session
	session.Mutex.Lock()
	// Update number of session values
	if _, exists := session.data[newPackage.Name]; !exists {
		format.Statistics.SessionValues++
	}
	session.data[newPackage.Name] = newPackage
	session.Mutex.Unlock()

	// Finish post processing
	go runHooks(newPackage)

	// Insert into database
	if influx.DB != nil {
		// Convert to a float if that suits the value, otherwise change field to value_string
		valueString := fmt.Sprintf("value=%s", newPackage.Value)
		if _, err := strconv.ParseFloat(newPackage.Value, 32); err != nil {
			valueString = fmt.Sprintf("value_string=\"%s\"", newPackage.Value)
		}

		// In Sessions, all values come in and out as strings regardless,
		// but this conversion alows Influx queries on the floats to be executed
		err := influx.DB.Write(fmt.Sprintf("%s value=%s", strings.Replace(newPackage.Name, " ", "_", -1), valueString))
		if err != nil {
			errorText := fmt.Sprintf("Error writing %s=%s to influx DB: %s", newPackage.Name, newPackage.Value, err.Error())
			// Only spam our log if Influx is online
			if influx.DB.Started {
				log.Error().Msg(errorText)
			}
			return fmt.Errorf(errorText)
		}
		log.Debug().Msgf("Logged %s=%s to database", newPackage.Name, newPackage.Value)
	}

	return nil
}
