package utils

import (
	"fmt"
	"strings"
	"unicode"
)

// FormatName returns string in upper case with underscores replacing spaces
func FormatName(name string) string {
	return strings.TrimSpace(strings.ToUpper(strings.Replace(name, " ", "_", -1)))
}

// IsValidName verifies the name is alphanumeric
func IsValidName(name string) bool {
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
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
