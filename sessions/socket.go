package sessions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/qcasey/MDroid-Core/format/response"
	"github.com/rs/zerolog/log"
)

// Session WebSocket upgrader
var upgrader = websocket.Upgrader{} // use default options

// GetSessionSocket returns the entire current session as a webstream
func GetSessionSocket(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true } // return true for now, although this should range over accepted origins

	// Log if requested
	log.Debug().Msg("Responding to request for session websocket")

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Msgf("Error upgrading webstream: %s", err.Error())
		return
	}
	defer c.Close()
	for {
		if _, _, err := c.ReadMessage(); err != nil {
			log.Error().Msgf("Error reading from webstream: %s", err.Error())
			break
		}

		// Pass through lock first
		writeSession := GetAllMin()
		if err = c.WriteJSON(writeSession); err != nil {
			log.Error().Msgf("Error writing to webstream: %s", err.Error())
			break
		}
	}
}

// checkServer will continiously ping a central server for waiting packets,
// and will open a websocket as a client if so
func checkServer(host string, token string) {
	var failedOnce bool
	log.Info().Msgf("Starting pings to main server at %s", host)

	for {
		// Start by assuming we're not on LTE, lower the wait time
		timeToWait := time.Millisecond * 500
		wifiConnected := GetBoolDefault("WIFI_CONNECTED", false)
		if !wifiConnected {
			// Slow frequency of pings while on LTE
			timeToWait = time.Second * 5
		}

		resp, err := http.Get(fmt.Sprintf("http://%s/ws/ping", host))
		if err != nil {
			// handle error
			SetValue("LAST_SERVER_RESPONSE", "ERROR")
			if !failedOnce {
				failedOnce = true
			} else {
				log.Error().Msgf("Error when pinging the central server.\n%s", err.Error())
			}
		} else {
			SetValue("LAST_SERVER_RESPONSE", resp.Status)
			resp.Body.Close()
			if resp.StatusCode == 200 {
				log.Info().Msg("Client is waiting on us, connect to server to acquire a websocket")
				runServerSocket(host, token)
			}
		}

		time.Sleep(timeToWait)
	}
}

func parseMessage(message []byte) *response.JSONResponse {
	resp := response.JSONResponse{}
	err := json.Unmarshal(message, &resp)
	if err != nil {
		log.Error().Msgf("Error marshalling json from websocket.\nJSON: %s\nError:%s", message, err.Error())
		return nil
	}

	// Check if the server is echoing back to us, or if it's a legitimate request from the server
	if resp.Method != "response" {
		output, ok := resp.Output.(string)
		if !ok {
			log.Error().Msg("Cannot cast output to string.")
			return nil
		}

		log.Info().Msgf("Websocket read request:  %s", output)
		internalResponse, path, err := getAPIResponse(output)
		if err != nil {
			log.Error().Msgf("Error from forwarded request websocket: %s", err.Error())
			return nil
		}

		log.Debug().Msgf("API response:  %s", string(internalResponse))
		response := response.JSONResponse{}
		err = json.Unmarshal(internalResponse, &response)
		if err != nil {
			log.Error().Msgf("Error marshalling response to websocket: %s", err.Error())
			return nil
		}
		response.Method = "response"
		response.Status = path
		return &response
	}
	return nil
}

func getAPIResponse(dataString string) ([]byte, string, error) {
	dataArray := strings.Split(dataString, ";")
	if len(dataArray) != 3 {
		const errMsg = "Could not break response into core components. Got response: %s"
		log.Error().Msgf(errMsg, dataString)
		return nil, "", fmt.Errorf(errMsg, dataString)
	}

	method := dataArray[0]
	path := dataArray[1]
	postingString := dataArray[2]

	var (
		resp *http.Response
		err  error
	)
	const errMsg = "Could not forward request from websocket. Got error: %s"

	if method == "POST" {
		jsonStr := []byte(postingString)
		req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:5353%s", path), bytes.NewBuffer(jsonStr))
		if err != nil {
			log.Error().Msgf(errMsg, err.Error())
			return nil, "", fmt.Errorf(errMsg, err.Error())
		}
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err = client.Do(req)
	} else if method == "GET" {
		resp, err = http.Get(fmt.Sprintf("http://localhost:5353%s", path))
	}

	if err != nil {
		log.Error().Msgf(errMsg, err.Error())
		return nil, "", fmt.Errorf(errMsg, err.Error())
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
	log.Info().Msgf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Error().Msgf("Error dialing websocket: %s", err.Error())
		return
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		err = c.WriteJSON(response.JSONResponse{Output: "Ready and willing.", Method: "response", Status: "success", OK: true})
		if err != nil {
			log.Error().Msgf("Error writing to websocket: %s", err.Error())
			return
		}

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Error().Msgf("Error reading from websocket.\nMessage: %s\nError:%s", message, err.Error())
				return
			}

			// parse message by newline, if necessary
			messageSplit := bytes.Split(message, []byte("\n"))
			if len(messageSplit) > 1 {
				log.Printf(fmt.Sprintf("Split incoming message into %d parts", len(messageSplit)))
			}
			for _, m := range messageSplit {
				log.Printf(fmt.Sprintf("Sending message %s", m))
				response := parseMessage(m)
				if response != nil {
					log.Info().Msgf("Responding with %v", *response)
					err = c.WriteJSON(*response)
					if err != nil {
						log.Error().Msgf("Error writing to websocket: %s", err.Error())
						return
					}
				}
			}
		}
	}()

	for {
		select {
		case <-done:
			log.Info().Msg("Closed websocket connection.")
			return
		}
	}
}
