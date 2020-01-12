package db

import (
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// Module begins module init
type Module struct{}

// Mod exports our Module implementation
var Mod *Module

// Setup parses this module's implementation
func (*Module) Setup(configAddr *map[string]string) {
	configMap := *configAddr
	databaseHost, usingDatabase := configMap["DATABASE_HOST"]
	if !usingDatabase {
		DB = nil
		log.Warn().Msg("Databases are disabled")
		return
	}

	databaseName, usingDatabase := configMap["DATABASE_NAME"]
	if !usingDatabase {
		DB = nil
		log.Warn().Msg("Databases are disabled")
		return
	}

	// Request to use SQLITE
	if databaseHost == "SQLITE" {
		DB = &Database{Host: databaseHost, DatabaseName: databaseName, Type: SQLite}
		dbname, err := DB.SQLiteInit()
		if err != nil {
			panic(err)
		}
		log.Info().Msgf("Using SQLite DB at %s", dbname)
		return
	}

	// Setup InfluxDB as normal
	DB = &Database{Host: databaseHost, DatabaseName: databaseName, Type: InfluxDB}
	log.Info().Msgf("Using InfluxDB at %s with DB name %s.", databaseHost, databaseName)
}

// SetRoutes implements MDroid module functions
func (*Module) SetRoutes(router *mux.Router) {
}
