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

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/db"
	"github.com/qcasey/MDroid-Core/format"
	"github.com/qcasey/MDroid-Core/format/response"
	"github.com/qcasey/MDroid-Core/mqtt"
	"github.com/qcasey/MDroid-Core/sessions/gps"
	"github.com/rs/zerolog/log"
)

// HandleSet updates or posts a new session value to the common session
func HandleSet(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)

	// Default to NOT OK response
	response := response.JSONResponse{OK: false}

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
		response.Write(&w, r)
		return
	}

	params := mux.Vars(r)
	var newdata Data

	if err = json.NewDecoder(r.Body).Decode(&newdata); err != nil {
		log.Error().Msgf("Error decoding incoming JSON:\n%s", err.Error())
		response.Output = err.Error()
		response.Write(&w, r)
		return
	}

	// Call the setter
	newdata.Name = params["name"]
	if err = Set(newdata); err != nil {
		response.Output = err.Error()
		response.Write(&w, r)
		return
	}

	// Craft OK response
	response.OK = true
	response.Output = newdata

	response.Write(&w, r)
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
	newPackage.date = time.Now().In(gps.GetTimezone())
	newPackage.LastUpdate = newPackage.date.Format("2006-01-02 15:04:05.999")

	// Correct name
	newPackage.Name = format.Name(newPackage.Name)

	// Trim off whitespace
	newPackage.Value = strings.TrimSpace(newPackage.Value)

	// Add / update value in global session after locking access to session
	session.Mutex.Lock()

	// Check if this is a new value we should insert into the DB
	oldPackage, exists := session.data[newPackage.Name]

	// Add new package to session
	session.data[newPackage.Name] = newPackage
	session.stats.Sets++
	addStat(newPackage)
	session.Mutex.Unlock()

	// Finish post processing
	go runHooks(newPackage)

	// Insert into database if this is a new/updated value
	if !exists || (exists && oldPackage.Value != newPackage.Value) {
		formattedName := strings.Replace(newPackage.Name, " ", "_", -1)

		if mqtt.IsConnected() {
			topic := fmt.Sprintf("session/%s", formattedName)
			go mqtt.Publish(topic, newPackage.Value)
		}

		if db.DB != nil {
			// Convert to a float if that suits the value, otherwise change field to value_string
			valueString := fmt.Sprintf("value=%s", newPackage.Value)
			if _, err := strconv.ParseFloat(newPackage.Value, 32); err != nil {
				valueString = fmt.Sprintf("value=\"%s\"", newPackage.Value)
			}

			// In Sessions, all values come in and out as strings regardless,
			// but this conversion alows Influx queries on the floats to be executed
			err := db.DB.Write(fmt.Sprintf("%s %s", formattedName, valueString))
			if err != nil {
				errorText := fmt.Sprintf("Error writing %s to database:\n%s", valueString, err.Error())
				// Only spam our log if Influx is online
				if db.DB.Started {
					log.Error().Msg(errorText)
				}
				return fmt.Errorf(errorText)
			}
		}
	}

	return nil
}
