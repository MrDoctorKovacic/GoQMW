package core

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/qcasey/viper"
	"github.com/rs/zerolog/log"
)

// Core is the struct of our publish / subscribe model
type Core struct {
	mutex       sync.RWMutex
	subscribers map[string][]chan Message

	Router    *mux.Router
	Settings  *viper.Viper
	Session   *viper.Viper
	StartTime time.Time
}

// Message containing arbitrary Content
type Message struct {
	Content   interface{}
	WriteDate time.Time // time it was inserted
}

// New creates a publish / subscribe interface for administering the program
func New(settingsFile string) *Core {
	core := &Core{
		Router:      mux.NewRouter(),
		Settings:    viper.New(),
		Session:     viper.New(),
		StartTime:   time.Now(),
		subscribers: make(map[string][]chan Message),
	}

	core.Settings.SetConfigName(settingsFile) // name of config file (without extension)
	core.Settings.AddConfigPath(".")          // optionally look for config in the working directory
	err := core.Settings.ReadInConfig()       // Find and read the config file
	if err != nil {
		log.Warn().Msg(err.Error())
	}
	core.Settings.WatchConfig()

	// Setup router
	core.injectRoutes()

	// Enable debugging from settings
	configureLogging(core.Settings.GetBool("mdroid.debug"))

	return core
}

// Subscribe will add the given channel as a listener to a topic
// Topic is expected to be compatible with a Viper selector
// Channel is expected to be buffered
func (core *Core) Subscribe(topic string, ch chan Message) {
	core.mutex.Lock()
	defer core.mutex.Unlock()

	core.subscribers[topic] = append(core.subscribers[topic], ch)
}

// Publish a given message to all subscribed entities
// Topic is expected to be compatible with a Viper selector
func (core *Core) Publish(topic string, m Message) {
	core.mutex.Lock()
	defer core.mutex.Unlock()

	splitTopic := strings.Split(topic, ".")
	isSetting := splitTopic[0] == "settings"
	key := strings.Join(splitTopic[1:], ".")

	// Set the data respectively
	if isSetting {
		core.addToSettings(key, m.Content)
	} else {
		core.addToSession(key, m.Content)
	}

	// Append time written
	m.WriteDate = time.Now()

	for _, ch := range core.subscribers[topic] {
		ch <- m
	}
}

func (core *Core) addToSession(key string, value interface{}) {
	oldKeyWrites := core.Session.GetInt(fmt.Sprintf("%s.writes", key))

	core.Session.Set(fmt.Sprintf("%s.value", key), value)
	core.Session.Set(fmt.Sprintf("%s.write_date", key), time.Now())
	core.Session.Set(fmt.Sprintf("%s.writes", key), oldKeyWrites+1)
}

func (core *Core) addToSettings(key string, value interface{}) {
	log.Info().Msgf("Updating setting of %s to %v", key, value)
	core.Settings.Set(key, value)

	// write to disk
	core.Settings.WriteConfig()
}
