package influx

import (
	"fmt"
	"log"

	"github.com/MrDoctorKovacic/MDroid-Core/external/status"
	"github.com/parnurzeal/gorequest"
)

// Influx for writing/posting/querying LocalHost db
type Influx struct {
	Host     string
	Database string
}

// InfluxStatus will control logging and reporting of status / warnings / errors
var InfluxStatus = status.NewStatus("Influx")

// Ping influx DB server for connectivity
func (db *Influx) Ping() error {
	// Ping db instance
	request := gorequest.New()
	resp, _, errs := request.Get(db.Host + "/ping").End()
	if errs != nil {
		InfluxStatus.Log(status.Error(), "Error opening JSON file on disk: "+errs[0].Error())
		log.Println("Errored: " + errs[0].Error())
		return errs[0]
	}

	InfluxStatus.Log(status.OK(), fmt.Sprintf("[Influx] Ping response: %d", resp.StatusCode))

	// Create Database if it doesn't exist
	db.CreateDatabase()

	return nil
}

// Write to influx DB server with data pairs
func (db *Influx) Write(msg string) error {
	//log.Println("Sending " + msg)
	request := gorequest.New()
	resp, body, errs := request.Post(db.Host + "/write?db=" + db.Database).Type("text").Send(msg).End()
	if errs != nil {
		InfluxStatus.Log(status.Error(), fmt.Sprintf("Error when writing to DB: %s/write?db=%s with message %s", db.Host, db.Database, msg))
		return errs[0]
	}

	if resp.StatusCode != 204 {
		InfluxStatus.Log(status.Warning(), fmt.Sprintf("Write/Post request response: %d", resp.StatusCode))
		InfluxStatus.Log(status.Warning(), "Received: "+body)
	}

	return nil
}

// Query to influx DB server with data pairs
func (db *Influx) Query(msg string) (string, error) {
	request := gorequest.New()
	resp, body, errs := request.Post(db.Host + "/query?db=" + db.Database).Type("text").Send("q=" + msg).End()
	if errs != nil {
		return "", errs[0]
	}

	InfluxStatus.Log(status.OK(), fmt.Sprintf("Query request response: %d", resp.StatusCode))
	InfluxStatus.Log(status.OK(), "Received: "+body)
	return body, nil
}

// ShowDatabases handles the creation of a missing log Database
func (db *Influx) ShowDatabases() (string, error) {
	request := gorequest.New()
	resp, body, errs := request.Get(db.Host + "/query?q=SHOW DATABASES").End()
	if errs != nil {
		return "", errs[0]
	}

	InfluxStatus.Log(status.OK(), fmt.Sprintf("Show Database request response: %d", resp.StatusCode))
	InfluxStatus.Log(status.OK(), "Received: "+body)
	return body, nil
}

// CreateDatabase handles the creation of a missing log Database
func (db *Influx) CreateDatabase() error {
	request := gorequest.New()
	resp, body, errs := request.Post(db.Host + "/query?q=CREATE DATABASE " + db.Database).End()
	if errs != nil {
		return errs[0]
	}

	if resp.StatusCode != 200 {
		InfluxStatus.Log(status.OK(), fmt.Sprintf("Create Database request response: %d", resp.StatusCode))
		InfluxStatus.Log(status.OK(), "Received: "+body)
	}
	return nil
}
