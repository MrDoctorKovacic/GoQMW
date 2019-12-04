package db

type databaseType int

const (
	// InfluxDB is a wrapper for InfluxDB functions
	InfluxDB databaseType = 0
	// SQLite is a wrapper for SQLite functions
	SQLite   databaseType = 0
)

// Database for writing/posting/querying db
type Database struct {
	Host         string
	DatabaseName string
	Type         databaseType
	Started      bool
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

// Transition wrappers for old influx or SQLite DBs

// Ping influx database server for connectivity
func (database *Database) Ping() (bool, error) {
	switch database.Type {
	case InfluxDB:
		database.InfluxPing()
	case SQLite:
		database.SQLitePing()
	}
}

// Insert will prepare a new write statement and pass it along
func (database *Database) Insert(measurement string, tags map[string]interface{}, fields map[string]interface{}) error {
	switch database.Type {
	case InfluxDB:
		database.InfluxInsert(measurement, tags, fields)
	case SQLite:
		database.SQLiteInsert(measurement, tags, fields)
	}
}

// Write to influx database server with data pairs
func (database *Database) Write(msg string) error {
	switch database.Type {
	case InfluxDB:
		database.InfluxWrite(msg)
	case SQLite:
		database.SQLiteWrite(msg)
	}
}
