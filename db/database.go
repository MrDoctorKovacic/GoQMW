package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

type databaseType int

const (
	// InfluxDB is a wrapper for InfluxDB functions
	InfluxDB databaseType = 0
	// SQLite is a wrapper for SQLite functions
	SQLite databaseType = 1
)

// Database for writing/posting/querying db
type Database struct {
	Host         string
	DatabaseName string
	Type         databaseType
	Started      bool

	sqlconn   *sql.DB
	sqlinsert *sql.Stmt
}

// DB currently being used
var DB *Database

// Helper function to parse interfaces as a DB string
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
func (database *Database) Insert(measurement string, tags map[string]interface{}, fields map[string]interface{}) error {
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
		return fmt.Errorf("Error writing %s to database:\n%s", writeString, err.Error())
	}

	// Debug log and return
	log.Debug().Msgf("Logged %s to database", stmt.String())
	return nil
}

// Transition wrappers for old influx or SQLite DBs

// Ping influx database server for connectivity
func (database *Database) Ping() (bool, error) {
	switch database.Type {
	case InfluxDB:
		return database.InfluxPing()
	case SQLite:
		return database.SQLitePing()
	}
	return false, nil
}

// Write to influx database server with data pairs
func (database *Database) Write(msg string) error {
	switch database.Type {
	case InfluxDB:
		return database.InfluxWrite(msg)
	case SQLite:
		return database.SQLiteWrite(msg)
	}
	return nil
}
