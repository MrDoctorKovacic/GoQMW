package logging

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
)

// SlackAlert sends a message to a slack channel webhook
func SlackAlert(channel string, message string) {
	if channel != "" {
		var jsonStr = []byte(fmt.Sprintf(`{"text":"%s"}`, message))
		req, err := http.NewRequest("POST", channel, bytes.NewBuffer(jsonStr))
		if err != nil {
			status.Log(Error(), err.Error())
			return
		}
		req.Header.Set("X-Custom-Header", "myvalue")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		status.Log(OK(), fmt.Sprintf("response Status: %s", resp.Status))
		status.Log(OK(), fmt.Sprintf("response Headers: %s", resp.Header))

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			status.Log(Error(), err.Error())
			return
		}
		status.Log(OK(), fmt.Sprintf("response Body: %s", string(body)))
	}
}
