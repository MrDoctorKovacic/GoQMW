package sessions

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/gorilla/websocket"
)

var clientConnected bool

// CheckServer will continiously ping a central server for waiting packets,
// and will open a websocket as a client if so
func (session *Session) CheckServer(host string, token string) {
	var timeToWait time.Duration
	for {
		lteEnabled, err := session.GetSessionValue("LTE_ON")
		if err != nil {
			SessionStatus.Log(logging.Error(), "Error getting LTE status.")
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
			SessionStatus.Log(logging.Error(), fmt.Sprintf("Error when pinging the central server.\n%s", err.Error()))
		} else {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				SessionStatus.Log(logging.OK(), "Client is waiting on us, connect to server to acquire a websocket")
				requestServerSocket(host, token)
			}
		}

		time.Sleep(timeToWait)
	}
}

func requestServerSocket(host string, token string) {
	// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
	// Use of this source code is governed by a BSD-style
	// license that can be found in the LICENSE file.
	u := url.URL{Scheme: "ws", Host: host, Path: fmt.Sprintf("/ws/%s", token)}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		SessionStatus.Log(logging.Error(), "Error dialing websocket: "+err.Error())
		return
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		clientConnected = true
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				SessionStatus.Log(logging.Error(), "Error reading from websocket: "+err.Error())
				return
			}
			SessionStatus.Log(logging.OK(), fmt.Sprintf("Websocket read:  %s"+string(message)))
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			clientConnected = false
			return
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				SessionStatus.Log(logging.Error(), "Error writing to websocket: "+err.Error())
				return
			}
		}
	}
}
