// Package influx is my own implementation of influxdb commands
package influx

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/parnurzeal/gorequest"
	"github.com/rs/zerolog/log"
)

// Influx for writing/posting/querying LocalHost db
type Influx struct {
	Host     string
	Database string
	Started  bool
}

// DB currently being used
var DB *Influx

// Ping influx DB server for connectivity
func (db *Influx) Ping() (bool, error) {
	// Ping db instance
	request := gorequest.New()
	resp, _, errs := request.Get(db.Host + "/ping").End()
	if errs != nil {
		return false, errs[0]
	}
	return resp.StatusCode == 204, nil
}

// Helper function to parse interfaces as an influx string
func parseWriterData(stmt *strings.Builder, data *map[string]interface{}) error {
	counter := 0
	for key, value := range *data {
		if counter > 0 {
			stmt.WriteString(",")
		}
		counter++

		// Parse based on data type
		switch vv := value.(type) {
		case bool:
			stmt.WriteString(fmt.Sprintf("%s=%v", key, vv))
		case string:
			stmt.WriteString(fmt.Sprintf("%s=\"%v\"", key, vv))
		case int:
			stmt.WriteString(fmt.Sprintf("%s=%d", key, int(vv)))
		case int64:
			stmt.WriteString(fmt.Sprintf("%s=%d", key, int(vv)))
		case float32:
			stmt.WriteString(fmt.Sprintf("%s=%f", key, float64(vv)))
		case float64:
			stmt.WriteString(fmt.Sprintf("%s=%f", key, float64(vv)))
		default:
			return fmt.Errorf("Cannot process type of %v", vv)
		}
	}
	return nil
}

// Insert will prepare a new write statement and pass it along
func (db *Influx) Insert(measurement string, tags map[string]interface{}, fields map[string]interface{}) error {
	if db == nil {
		return fmt.Errorf("Database is nil")
	}

	// Prepare new insert statement
	var stmt strings.Builder
	stmt.WriteString(measurement)

	// Write tags first
	var tagstring strings.Builder
	if err := parseWriterData(&tagstring, &tags); err != nil {
		return err
	}

	// Check if any tags were added. If not, remove the trailing comma
	if tagstring.String() != "" {
		stmt.WriteRune(',')
	}

	// Space between tags and fields
	stmt.WriteString(tagstring.String())
	stmt.WriteRune(' ')

	// Write fields next
	if err := parseWriterData(&stmt, &fields); err != nil {
		return err
	}

	writeString := stmt.String()

	// Pass string we've built to write function
	if err := db.Write(writeString); err != nil {
		return fmt.Errorf("Error writing %s to influx DB:\n%s", writeString, err.Error())
	}

	// Debug log and return
	log.Debug().Msgf("Logged %s to database", stmt.String())
	return nil
}

// Write to influx DB server with data pairs
func (db *Influx) Write(msg string) error {
	// Check for positive ping response first.
	if !db.Started {
		if isOnline, err := db.Ping(); !isOnline {
			if err != nil {
				return err
			}
			return nil
		}
		db.Started = true
	}

	request := gorequest.New()
	resp, _, errs := request.Post(db.Host + "/write?db=" + db.Database).Type("text").Send(msg).End()
	if errs != nil {
		return errs[0]
	}

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("InfluxDB write failed with code %d. Request: %s \nResponse body: %s", resp.StatusCode, request.Data, body)
	}

	return nil
}

// Query to influx DB server with data pairs
func (db *Influx) Query(msg string) (string, error) {
	request := gorequest.New()
	_, body, errs := request.Post(db.Host + "/query?db=" + db.Database).Type("text").Send("q=" + msg).End()
	if errs != nil {
		return "", errs[0]
	}

	return body, nil
}

// ShowDatabases handles the creation of a missing log Database
func (db *Influx) ShowDatabases() (string, error) {
	request := gorequest.New()
	_, body, errs := request.Get(db.Host + "/query?q=SHOW DATABASES").End()
	if errs != nil {
		return "", errs[0]
	}

	return body, nil
}

// CreateDatabase handles the creation of a missing log Database
func (db *Influx) CreateDatabase() error {
	request := gorequest.New()
	_, _, errs := request.Post(db.Host + "/query?q=CREATE DATABASE " + db.Database).End()
	if errs != nil {
		return errs[0]
	}

	return nil
}
