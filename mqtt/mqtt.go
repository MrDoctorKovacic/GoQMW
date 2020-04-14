package mqtt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	logger "github.com/rs/zerolog/log"
)

// Module exports MDroid module
type Module struct{}

// config holds configuration and status of MQTT
type config struct {
	address  string
	clientid string
	username string
	password string
}

type message struct {
	Method   string `json:"method,omitempty"`
	Path     string `json:"path,omitempty"`
	PostData string `json:"postData,omitempty"`
}

var (
	// Mod exports our module functionality
	Mod Module

	mqttConfig    config
	finishedSetup bool
	client        mqtt.Client
)

var f mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	logger.Info().Msgf("TOPIC: %s\n", msg.Topic())
	logger.Info().Msgf("MSG: %s\n", msg.Payload())

	request := message{}
	err := json.Unmarshal(msg.Payload(), &request)

	var response *http.Response
	const errMsg = "Could not forward request from websocket. Got error: %s"

	if request.Method == "POST" {
		jsonStr := []byte(request.PostData)
		req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:5353%s", request.Path), bytes.NewBuffer(jsonStr))
		if err != nil {
			logger.Error().Msgf(errMsg, err.Error())
			return
		}
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		response, err = client.Do(req)
	} else if request.Method == "GET" {
		response, err = http.Get(fmt.Sprintf("http://localhost:5353%s", request.Path))
	}

	if err != nil {
		logger.Error().Msgf(errMsg, err.Error())
		return
	}

	defer response.Body.Close()
	return
}

// Publish will write the given message to the given topic and wait
func Publish(topic string, message string) {
	if !IsConnected() {
		connect()
	}
	token := client.Publish(fmt.Sprintf("vehicle/%s", topic), 0, true, message)
	token.Wait()
}

// IsConnected returns if the MQTT client has finished setting up and is connected
func IsConnected() bool {
	if !finishedSetup || client == nil {
		return false
	}

	return client.IsConnected()
}

func connect() {
	finishedSetup = false
	mqtt.DEBUG = log.New(os.Stdout, "", 0)
	mqtt.ERROR = log.New(os.Stdout, "", 0)
	logger.Info().Msgf("%s %s %s %s", mqttConfig.address, mqttConfig.clientid, mqttConfig.username, mqttConfig.password)
	opts := mqtt.NewClientOptions().AddBroker(mqttConfig.address).SetClientID(mqttConfig.clientid)
	opts.SetUsername(mqttConfig.username)
	opts.SetPassword(mqttConfig.password)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetDefaultPublishHandler(f)
	opts.SetPingTimeout(15 * time.Second)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		logger.Error().Msg(token.Error().Error())
		return
	}

	if token := client.Subscribe("vehicle/requests/#", 0, nil); token.Wait() && token.Error() != nil {
		logger.Error().Msg(token.Error().Error())
		return
	}
	finishedSetup = true
}

// Setup handles module init
func (*Module) Setup(configAddr *map[string]string) {
	configMap := *configAddr

	var ok bool
	mqttConfig.address, ok = configMap["MQTT_ADDRESS"]
	if !ok {
		logger.Warn().Msgf("Missing MQTT address.")
		return
	}
	mqttConfig.clientid, ok = configMap["MQTT_CLIENT_ID"]
	if !ok {
		logger.Warn().Msgf("Missing MQTT client ID.")
		return
	}
	mqttConfig.username, ok = configMap["MQTT_USERNAME"]
	if !ok {
		logger.Warn().Msgf("Missing MQTT username.")
		return
	}
	mqttConfig.password, ok = configMap["MQTT_PASSWORD"]
	if !ok {
		logger.Warn().Msgf("Missing MQTT password.")
		return
	}

	go connect()
}
