package db

import (
	"database/sql"
	"fmt"
	"time"

	// For SQLite functionality
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
)

// SQLiteInit creates a new SQLite connection
func (database *Database) SQLiteInit() (string, error) {
	var err error
	database.mutex.Lock()
	defer database.mutex.Unlock()

	// TODO: make this a setting
	filename := fmt.Sprintf("/home/pi/MDroid/logs/core/dbs/%s.db", time.Now().Local().String())

	database.sqlconn, err = sql.Open("sqlite3", filename)
	if err != nil {
		return filename, err
	}
	statement, err := database.sqlconn.Prepare("CREATE TABLE IF NOT EXISTS vehicle (id INTEGER PRIMARY KEY, timestamp INTEGER, msg TEXT)")
	if err != nil {
		return filename, err
	}
	statement.Exec()
	statement.Close()
	database.Started = true

	database.sqlinsert, err = database.sqlconn.Prepare("INSERT INTO vehicle (timestamp, msg) VALUES (?, ?)")
	return filename, nil
}

// SQLitePing database server for connectivity
func (database *Database) SQLitePing() (bool, error) {
	// Ping database instance
	return database.sqlconn != nil, nil
}

// SQLiteWrite to SQLite database server with data pairs
func (database *Database) SQLiteWrite(msg string) error {
	// Check for positive ping response first.
	if !database.Started {
		log.Info().Msg("DB is closed, reopening...")
		database.SQLiteInit()
	}

	_, err := database.sqlinsert.Exec(time.Now().Local().Nanosecond(), msg)
	if err != nil {
		return err
	}
	return nil
}
