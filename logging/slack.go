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
		req, _ := http.NewRequest("POST", channel, bytes.NewBuffer(jsonStr))
		req.Header.Set("X-Custom-Header", "myvalue")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		fmt.Println("response Status:", resp.Status)
		fmt.Println("response Headers:", resp.Header)
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("response Body:", string(body))
	}
}
