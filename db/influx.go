// Package db is my own implementation of influxdatabase commands
package db

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/parnurzeal/gorequest"
	"github.com/rs/zerolog/log"
)

// InfluxPing database server for connectivity
func (database *Database) InfluxPing() (bool, error) {
	// Ping database instance
	request := gorequest.New()
	resp, _, errs := request.Get(database.Host + "/ping").End()
	if errs != nil {
		return false, errs[0]
	}
	return resp.StatusCode == 204, nil
}

// InfluxInsert will prepare a new write statement and pass it along
func (database *Database) InfluxInsert(measurement string, tags map[string]interface{}, fields map[string]interface{}) error {
	if database == nil {
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
	if err := database.Write(writeString); err != nil {
		return fmt.Errorf("Error writing %s to influx database:\n%s", writeString, err.Error())
	}

	// Debug log and return
	log.Debug().Msgf("Logged %s to database", stmt.String())
	return nil
}

// InfluxWrite to influx database server with data pairs
func (database *Database) InfluxWrite(msg string) error {
	// Check for positive ping response first.
	if !database.Started {
		if isOnline, err := database.Ping(); !isOnline {
			if err != nil {
				return err
			}
			return nil
		}
		database.Started = true
	}

	request := gorequest.New()
	resp, _, errs := request.Post(database.Host + "/write?database=" + database.DatabaseName).Type("text").Send(msg).End()
	if errs != nil {
		return errs[0]
	}

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("Influxdatabase write failed with code %d. Request: %s \nResponse body: %s", resp.StatusCode, request.Data, body)
	}

	return nil
}

// InfluxQuery to influx database server with data pairs
func (database *Database) InfluxQuery(msg string) (string, error) {
	request := gorequest.New()
	_, body, errs := request.Post(database.Host + "/query?database=" + database.DatabaseName).Type("text").Send("q=" + msg).End()
	if errs != nil {
		return "", errs[0]
	}

	return body, nil
}

// Query to influx database server with data pairs
func (database *Database) Query(msg string) (string, error) {
	request := gorequest.New()
	_, body, errs := request.Post(database.Host + "/query?database=" + database.DatabaseName).Type("text").Send("q=" + msg).End()
	if errs != nil {
		return "", errs[0]
	}

	return body, nil
}

// ShowDatabases handles the creation of a missing log Database
func (database *Database) ShowDatabases() (string, error) {
	request := gorequest.New()
	_, body, errs := request.Get(database.Host + "/query?q=SHOW DATABASES").End()
	if errs != nil {
		return "", errs[0]
	}

	return body, nil
}

// CreateDatabase handles the creation of a missing log Database
func (database *Database) CreateDatabase() error {
	request := gorequest.New()
	_, _, errs := request.Post(database.Host + "/query?q=CREATE DATABASE " + database.DatabaseName).End()
	if errs != nil {
		return errs[0]
	}

	return nil
}
