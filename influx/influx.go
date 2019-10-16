// Package influx is my own implementation of influxdb commands
package influx

import (
	"github.com/parnurzeal/gorequest"
)

// Influx for writing/posting/querying LocalHost db
type Influx struct {
	Host     string
	Database string
	Started  bool
}

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

// Write to influx DB server with data pairs
func (db *Influx) Write(msg string) (bool, error) {

	// Check for positive ping response first.
	// Throw away these requests, since they're being saved in session & will
	// be outdated by the time Influx wakes up
	if !db.Started {
		if isOnline, err := db.Ping(); !isOnline {
			if err != nil {
				return false, err
			}
			return false, nil
		}
		db.Started = true
	}

	request := gorequest.New()
	_, _, errs := request.Post(db.Host + "/write?db=" + db.Database).Type("text").Send(msg).End()
	if errs != nil {
		return true, errs[0]
	}

	return true, nil
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
