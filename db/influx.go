// Package db is my own implementation of influxdatabase commands
package db

import (
	"fmt"
	"io/ioutil"

	"github.com/parnurzeal/gorequest"
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
	url := database.Host + "/write?database=" + database.DatabaseName
	resp, _, errs := request.Post(url).Type("text").Send(msg).End()
	if errs != nil {
		return errs[0]
	}

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("Write to database %s failed with code %d.\nRequest: %s\nRequest Body: %s\nResponse body: %s", database.DatabaseName, resp.StatusCode, url, msg, body)
	}

	return nil
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
