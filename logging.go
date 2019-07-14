package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"
)

var debugLog map[string]map[string]string
var debugLock sync.Mutex
var debugFile string

// writeLog to given file, create one if it doesn't exist
func writeLog(file string) error {
	debugLock.Lock()
	defer debugLock.Unlock()
	debugJSON, err := json.Marshal(debugLog)

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(file, debugJSON, 0644)
	if err != nil {
		return err
	}

	return nil
}

func EnableLogging(debugFilename string) (bool, error) {
	file, err := os.Open(debugFilename)
	defer file.Close()
	if err != nil {
		return false, err
	}

	// Init global variables
	debugLog = make(map[string]map[string]string)
	debugFile = debugFilename
	return true, nil
}

// LoggingMiddleware will generate a file for reproducing a live session, for debug purposes
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var timestamp = time.Now().In(TIMEZONE).Format(time.RFC850)
		data, _ := ioutil.ReadAll(r.Body)
		r.Body.Close()
		r.Body = ioutil.NopCloser(bytes.NewBuffer(data))

		// Add route to log, after getting lock
		debugLock.Lock()
		debugLog[timestamp] = make(map[string]string, 0)
		debugLog[timestamp]["REQUEST_URI"] = r.RequestURI
		debugLog[timestamp]["REQUEST_DATA"] = string(data)
		debugLock.Unlock()

		// Write out to file
		if debugFile != "" {
			writeLog(debugFile)
		}
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}
