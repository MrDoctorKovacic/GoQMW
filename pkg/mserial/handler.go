package mserial

import (
	"sync"
	"time"

	"github.com/qcasey/MDroid-Core/internal/core"
	"github.com/rs/zerolog/log"
	"github.com/tarm/serial"
)

// Message for the serial writer, and a channel to await it
type Message struct {
	Device     *serial.Port
	Text       string
	isComplete chan error
	UUID       string
}

var (
	// Writer is our one main port to default to
	Writer         *serial.Port
	writeQueue     map[*serial.Port][]*Message
	writeQueueLock sync.Mutex
)

func init() {
	writeQueue = make(map[*serial.Port][]*Message, 0)
}

// Setup handles module init
/*
func Setup(c *core.Core) {
	if !c.Settings.IsSet("mdroid.HARDWARE_SERIAL_PORT") {
		log.Warn().Msgf("No hardware serial port defined. Not setting up serial devices.")
		return
	}

	hardwareSerialPort := c.Settings.GetString("mdroid.HARDWARE_SERIAL_PORT")

	// Start initial reader / writer
	log.Info().Msgf("Registering %s as serial writer", hardwareSerialPort)
	go startSerialComms( hardwareSerialPort, 115200)

	// Setup other devices
	/*
		for device, baudrate := range parseSerialDevices(settings.GetAll()) {
			go startSerialComms(device, baudrate)
		}*/

//
// Serial routes
//
//}

// Start will set up the serial port and ReadSerial goroutine
func Start(c *core.Core, deviceName string, baudrate int) {
	s, err := openSerialPort(deviceName, baudrate)
	if err != nil {
		log.Error().Msgf("Failed to open serial port %s", deviceName)
		log.Error().Msg(err.Error())
		time.Sleep(time.Second * 2)
		go Start(c, deviceName, baudrate)
		return
	}
	defer s.Close()

	// Use first Serial device as a R/W, all others will only be read from
	isWriter := false
	if Writer == nil {
		Writer = s
		isWriter = true
		log.Info().Msgf("Using serial device %s as default writer", deviceName)
	}

	// Continually read from serial port
	log.Info().Msgf("Starting new serial reader on device %s", deviceName)
	loop(s, isWriter) // this will block until abrubtly ended
	log.Error().Msg("Serial disconnected, closing port and reopening in 10 seconds")

	// Replace main serial writer
	if Writer == s {
		Writer = nil
	}

	s.Close()
	time.Sleep(time.Second * 10)
	log.Error().Msg("Reopening serial port...")
	go Start(c, deviceName, baudrate)
}

// Push queues a message for writing
func Push(m *Message) {
	writeQueueLock.Lock()
	defer writeQueueLock.Unlock()
	_, ok := writeQueue[m.Device]
	if !ok {
		writeQueue[m.Device] = []*Message{}
	}

	writeQueue[m.Device] = append(writeQueue[m.Device], m)
}

// PushText creates a new message with the default writer, and appends it for sending
func PushText(message string) {
	m := &Message{Device: Writer, Text: message}
	Push(m)
}

// Await queues a message for writing, and waits for it to be sent
func Await(m *Message) error {
	//m.UUID, _ = format.NewShortUUID()
	m.isComplete = make(chan error)
	//log.Info().Msgf("[%s] Awaiting serial message write", m.UUID)
	Push(m)
	err := <-m.isComplete
	//log.Info().Msgf("[%s] Message write is complete", m.UUID)
	return err
}

// AwaitText creates a new message with the default writer, appends it for sending, and waits for it to be sent
func AwaitText(message string) error {
	//uuid, _ := format.NewShortUUID()
	m := &Message{Device: Writer, Text: message, isComplete: make(chan error)}
	//log.Info().Msgf("[%s] Awaiting serial message write", m.UUID)
	Push(m)
	err := <-m.isComplete
	//log.Info().Msgf("[%s] Message write is complete", m.UUID)
	return err
}

// Pop the last message off the queue and write it to the respective serial
func Pop(device *serial.Port) {
	if device == nil {
		log.Error().Msg("Serial port is not set, nothing to write to.")
		return
	}

	writeQueueLock.Lock()
	defer writeQueueLock.Unlock()
	queue, ok := writeQueue[device]
	if !ok || len(queue) == 0 {
		return
	}

	var msg *Message
	msg, writeQueue[device] = writeQueue[device][len(writeQueue[device])-1], writeQueue[device][:len(writeQueue[device])-1]

	err := write(msg)
	msg.isComplete <- err
}
