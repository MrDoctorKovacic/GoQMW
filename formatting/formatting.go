// Package formatting are common utilities used across the MDroid suite
package formatting

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// JSONResponse for common return value to API
type JSONResponse struct {
	Output interface{} `json:"output,omitempty"`
	Status string      `json:"status,omitempty"`
	OK     bool        `json:"ok,omitempty"`
	Method string      `json:"method,omitempty"`
	ID     int         `json:"id,omitempty"`
}

// FormatName returns string in upper case with underscores replacing spaces
func FormatName(name string) string {
	return strings.TrimSpace(strings.ToUpper(strings.Replace(name, " ", "_", -1)))
}

// IsValidName verifies the name is alphanumeric
func IsValidName(name string) bool {
	for _, r := range name {
		if string(r) != "_" && !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			return false
		}
	}
	return true
}

// IsPositiveRequest helps translate UP or LOCK into true or false
func IsPositiveRequest(request string) (bool, error) {
	switch request {
	case "ON":
		fallthrough
	case "UP":
		fallthrough
	case "LOCK":
		fallthrough
	case "OPEN":
		fallthrough
	case "TOGGLE":
		fallthrough
	case "PUSH":
		return true, nil

	case "OFF":
		fallthrough
	case "DOWN":
		fallthrough
	case "UNLOCK":
		fallthrough
	case "CLOSE":
		return false, nil
	}

	// Command didn't match any of the above, get out of here
	return false, fmt.Errorf("Error: %s in an invalid command", request)
}

// CompareTimestamps assuming both timezones are the same
func CompareTimestamps(time1 string, time2 string) (time.Duration, error) {
	t, err := time.Parse("2006-01-02 15:04:05.999", time2)
	t2, err2 := time.Parse("2006-01-02 15:04:05.999", time2)

	if err != nil {
		return 0, err
	}
	if err2 != nil {
		return 0, err2
	}

	return t.Sub(t2), nil
}

// CompareTimeToNow given a properly formatted time and timezone
func CompareTimeToNow(time1 string, timezone *time.Location) (time.Duration, error) {
	t, err := time.Parse("2006-01-02 15:04:05.999", time1)

	if err != nil {
		return 0, err
	}
	return time.Now().In(timezone).Sub(t), nil
}
