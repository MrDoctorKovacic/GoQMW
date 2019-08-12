package utils

import "strings"

// FormatName returns string in upper case with underscores replacing spaces
func FormatName(text string) string {
	return strings.ToUpper(strings.Replace(text, " ", "_", -1))
}
