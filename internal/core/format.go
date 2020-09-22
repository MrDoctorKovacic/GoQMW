package core

import (
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func configureLogging(debug bool) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.CallerMarshalFunc = func(file string, line int) string {
		fileparts := strings.Split(file, "/")
		filename := strings.Replace(fileparts[len(fileparts)-1], ".go", "", -1)
		return filename + ":" + strconv.Itoa(line)
	}
	zerolog.TimeFieldFormat = "3:04PM"
	output := zerolog.ConsoleWriter{Out: os.Stderr}
	log.Logger = zerolog.New(output).With().Timestamp().Caller().Logger()
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

// FormatName will keep a key consistient throughout MDroid core
func FormatName(name string) string {
	spaceRemover := regexp.MustCompile(`\s+`)
	name = spaceRemover.ReplaceAllString(name, " ")
	return strings.ToUpper(strings.Replace(strings.TrimSpace(name), " ", "_", -1))
}
