package sessions

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/rs/zerolog/log"
)

// SlackAlert sends a message to a slack channel webhook
func SlackAlert(channel string, message string) {
	if channel == "" {
		log.Warn().Msg("Empty slack channel")
		return
	}
	if message == "" {
		log.Warn().Msg("Empty slack message")
		return
	}

	jsonStr := []byte(fmt.Sprintf(`{"text":"%s"}`, message))
	req, err := http.NewRequest("POST", channel, bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Error().Msg(err.Error())
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

	log.Info().Msg(fmt.Sprintf("response Status: %s", resp.Status))
	log.Info().Msg(fmt.Sprintf("response Headers: %s", resp.Header))

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}
	log.Info().Msg(fmt.Sprintf("response Body: %s", string(body)))
}
