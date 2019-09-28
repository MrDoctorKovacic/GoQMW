package sessions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/gorilla/websocket"
)

var clientConnected bool

// CheckServer will continiously ping a central server for waiting packets,
// and will open a websocket as a client if so
func CheckServer(host string, token string) {
	var timeToWait time.Duration
	for {
		if !clientConnected {
			lteEnabled, err := Get("LTE_ON")
			if err != nil {
				status.Log(logging.Error(), "Error getting LTE status.")
				timeToWait = time.Second * 5
			} else if lteEnabled.Value == "TRUE" {
				// Slow frequency of pings while on LTE
				timeToWait = time.Second * 5
			} else {
				// We can assume we're not on LTE, lower the wait time
				timeToWait = time.Second * 1
			}

			resp, err := http.Get(fmt.Sprintf("http://%s/ws/ping", host))
			if err != nil {
				// handle error
				status.Log(logging.Error(), fmt.Sprintf("Error when pinging the central server.\n%s", err.Error()))
			} else {
				resp.Body.Close()
				if resp.StatusCode == 200 {
					status.Log(logging.OK(), "Client is waiting on us, connect to server to acquire a websocket")
					runServerSocket(host, token)
				}
			}
		}

		time.Sleep(timeToWait)
	}
}

func getAPIResponse(dataString string) ([]byte, string, error) {
	dataArray := strings.Split(dataString, ";")
	if len(dataArray) != 3 {
		status.Log(logging.Error(), fmt.Sprintf("Could not break response into core components. Got response: %s", dataString))
		return nil, "", fmt.Errorf("Could not break response into core components. Got response: %s", dataString)
	}

	method := dataArray[0]
	path := dataArray[1]
	postingString := dataArray[2]

	var (
		resp *http.Response
		err  error
	)
	if method == "POST" {
		jsonStr := []byte(postingString)
		req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:5353%s", path), bytes.NewBuffer(jsonStr))
		if err != nil {
			status.Log(logging.Error(), fmt.Sprintf("Could not forward request from websocket. Got error: %s", err.Error()))
			return nil, "", fmt.Errorf("Could not forward request from websocket. Got error: %s", err.Error())
		}
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err = client.Do(req)
	} else if method == "GET" {
		resp, err = http.Get(fmt.Sprintf("http://localhost:5353%s", path))
	}

	if err != nil {
		status.Log(logging.Error(), fmt.Sprintf("Could not forward request from websocket. Got error: %s", err.Error()))
		return nil, "", fmt.Errorf("Could not forward request from websocket. Got error: %s", err.Error())
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return body, path, nil
}

func runServerSocket(host string, token string) {
	// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
	// Use of this source code is governed by a BSD-style
	// license that can be found in the LICENSE file.
	u := url.URL{Scheme: "ws", Host: host, Path: fmt.Sprintf("/ws/%s", token)}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		status.Log(logging.Error(), "Error dialing websocket: "+err.Error())
		return
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		clientConnected = true
		defer close(done)
		err = c.WriteJSON(formatting.JSONResponse{Output: "Ready and willing.", Method: "response", Status: "success", OK: true})
		if err != nil {
			status.Log(logging.Error(), "Error writing to websocket: "+err.Error())
			return
		}

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				status.Log(logging.Error(), "Error reading from websocket: "+err.Error())
				return
			}
			response := formatting.JSONResponse{}
			err = json.Unmarshal(message, &response)

			if err != nil {
				status.Log(logging.Error(), "Error marshalling json from websocket: "+err.Error())
				return
			}

			// Check if the server is echoing back to us, or if it's a legitimate request from the server
			if response.Method != "response" {
				// TODO! Match this path against a walk through of our router
				//output := fmt.Sprintf("%v", response.Output)
				output, ok := response.Output.(string)
				if !ok {
					status.Log(logging.Error(), "Cannot cast output to string.")
					return
				}

				status.Log(logging.OK(), fmt.Sprintf("Websocket read output:  %s", output))
				internalResponse, path, err := getAPIResponse(output)
				if err != nil {
					status.Log(logging.Error(), "Error from forwarded request websocket: "+err.Error())
					return
				}

				status.Log(logging.OK(), fmt.Sprintf("Internal API response:  %s", string(internalResponse)))
				response := formatting.JSONResponse{}
				err = json.Unmarshal(internalResponse, &response)
				if err != nil {
					status.Log(logging.Error(), "Error marshalling response to websocket: "+err.Error())
					return
				}
				response.Method = "response"
				response.Status = path

				err = c.WriteJSON(response)
				if err != nil {
					status.Log(logging.Error(), "Error writing to websocket: "+err.Error())
					return
				}

			}
		}
	}()

	for {
		select {
		case <-done:
			clientConnected = false
			status.Log(logging.OK(), "Closed websocket connection.")
			return
		}
	}
}
