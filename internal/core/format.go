package core

import (
	"regexp"
	"strings"
)

func FormatName(name string) string {
	spaceRemover := regexp.MustCompile(`\s+`)
	name = spaceRemover.ReplaceAllString(name, " ")
	return strings.ToUpper(strings.Replace(strings.TrimSpace(name), " ", "_", -1))
}
