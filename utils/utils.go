package utils

import "strings"

// Formats in upper case with underscores replacing spaces
func formatName(text string) string {
	return strings.ToUpper(strings.Replace(text, " ", "_", -1))
}
