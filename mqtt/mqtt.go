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
	address         string
	addressFallback string
	clientid        string
	username        string
	password        string
}

type message struct {
	Method   string `json:"method,omitempty"`
	Path     string `json:"path,omitempty"`
	PostData string `json:"postData,omitempty"`
}

var (
	// Mod exports our module functionality
	Mod Module

	// Enabled if MQTT is enabled
	Enabled bool

	mqttConfig    config
	finishedSetup bool
	remoteClient  mqtt.Client
	localClient   mqtt.Client
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
		httpClient := &http.Client{}
		response, err = httpClient.Do(req)
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
func Publish(topic string, message interface{}, publishToRemote bool) {
	timesSlept := 0
	for !IsReady() || !IsConnected() {
		time.Sleep(500 * time.Millisecond)
		if timesSlept > 0 && timesSlept%60 == 0 {
			logger.Warn().Msgf("Has waited %d seconds to get this packet out, still not connected", timesSlept/2)
		}
		timesSlept++
	}
	localToken := localClient.Publish(fmt.Sprintf("vehicle/%s", topic), 0, true, message)

	if publishToRemote {
		remoteToken := remoteClient.Publish(fmt.Sprintf("vehicle/%s", topic), 0, true, message)
		remoteToken.Wait()
	}

	localToken.Wait()
}

// IsReady returns if the MQTT client has finished setting up
func IsReady() bool {
	return finishedSetup && remoteClient != nil && localClient != nil
}

// IsConnected returns if the MQTT client is connected
func IsConnected() bool {
	return remoteClient.IsConnected() && localClient.IsConnected()
}

func checkReconnection() {
	for {
		if IsReady() && !IsConnected() {
			logger.Error().Msg("MQTT connection lost... retrying..")
			connect()
		}
		time.Sleep(1500 * time.Millisecond)
	}
}

func reconnect() {
	go func() {
		logger.Error().Msg("Failed to setup MQTT, waiting half a second and retrying..")
		time.Sleep(500 * time.Millisecond)
		connect()
	}()
}

func connect() {
	finishedSetup = false
	//mqtt.DEBUG = log.New(os.Stdout, "", 0)
	mqtt.ERROR = log.New(os.Stdout, "", 0)

	// Remote Client
	opts := mqtt.NewClientOptions().AddBroker(mqttConfig.address).SetClientID(mqttConfig.clientid)
	opts.SetCleanSession(false)
	opts.SetMaxReconnectInterval(30 * time.Second)
	opts.SetUsername(mqttConfig.username)
	opts.SetPassword(mqttConfig.password)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetDefaultPublishHandler(f)
	opts.SetPingTimeout(15 * time.Second)

	remoteClient = mqtt.NewClient(opts)
	if token := remoteClient.Connect(); token.Wait() && token.Error() != nil {
		logger.Error().Msg(token.Error().Error())
		reconnect()
		return
	}
	if token := remoteClient.Subscribe("vehicle/requests/#", 0, nil); token.Wait() && token.Error() != nil {
		logger.Error().Msg(token.Error().Error())
		reconnect()
		return
	}

	// Local Client
	opts = mqtt.NewClientOptions().AddBroker(mqttConfig.addressFallback).SetClientID(mqttConfig.clientid).SetAutoReconnect(true)
	opts.SetCleanSession(false)
	opts.SetMaxReconnectInterval(3 * time.Second)
	opts.SetUsername(mqttConfig.username)
	opts.SetPassword(mqttConfig.password)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetDefaultPublishHandler(f)
	opts.SetPingTimeout(15 * time.Second)

	localClient = mqtt.NewClient(opts)
	if token := localClient.Connect(); token.Wait() && token.Error() != nil {
		logger.Error().Msg(token.Error().Error())
		reconnect()
		return
	}
	if token := localClient.Subscribe("vehicle/requests/#", 0, nil); token.Wait() && token.Error() != nil {
		logger.Error().Msg(token.Error().Error())
		reconnect()
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
	mqttConfig.addressFallback, ok = configMap["MQTT_ADDRESS_FALLBACK"]
	if !ok {
		logger.Warn().Msgf("Missing MQTT fallback address.")
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

	Enabled = true
	go connect()
	go checkReconnection()
}
