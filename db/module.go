package db

import (
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Module begins module init
type Module struct{}

// Mod exports our Module implementation
var Mod *Module

// Setup parses this module's implementation
func (*Module) Setup() {
	if !viper.IsSet("mdroid.DATABASE_HOST") || !viper.IsSet("mdroid.DATABASE_NAME") {
		DB = nil
		log.Warn().Msg("Databases are disabled")
		return
	}

	databaseHost := viper.GetString("mdroid.DATABASE_HOST")
	databaseName := viper.GetString("mdroid.DATABASE_NAME")

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
