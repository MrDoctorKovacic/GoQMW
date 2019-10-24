package bluetooth

import (
	"errors"
	"fmt"

	"github.com/muka/go-bluetooth/api"
	"github.com/muka/go-bluetooth/bluez/profile/adapter"
	"github.com/muka/go-bluetooth/bluez/profile/device"
	"github.com/rs/zerolog/log"
)

func client(adapterID, hwaddr string) (err error) {

	log.Info().Msg(fmt.Sprintf("Discovering %s on %s", hwaddr, adapterID))

	a, err := adapter.NewAdapter1FromAdapterID(adapterID)
	if err != nil {
		return err
	}

	dev, err := discover(a, hwaddr)
	if err != nil {
		return err
	}

	if dev == nil {
		return errors.New("Device not found, is it advertising?")
	}

	err = connect(dev)
	if err != nil {
		return err
	}

	watchProps, err := dev.WatchProperties()
	if err != nil {
		return err
	}

	for propUpdate := range watchProps {
		log.Debug().Msg(fmt.Sprintf("propUpdate %++v", propUpdate))
	}

	return nil
}

func discover(a *adapter.Adapter1, hwaddr string) (*device.Device1, error) {

	err := a.FlushDevices()
	if err != nil {
		return nil, err
	}

	discovery, cancel, err := api.Discover(a, nil)
	if err != nil {
		return nil, err
	}

	defer cancel()

	for ev := range discovery {

		dev, err1 := device.NewDevice1(ev.Path)
		if err != nil {
			return nil, err1
		}

		if dev == nil || dev.Properties == nil {
			continue
		}

		p := dev.Properties

		n := p.Alias
		if p.Name != "" {
			n = p.Name
		}
		log.Debug().Msg(fmt.Sprintf("Discovered (%s) %s", n, p.Address))

		if p.Address != hwaddr {
			continue
		}

		log.Info().Msg(fmt.Sprintf("Found device %s", p.Address))
		return dev, nil
	}

	return nil, nil
}

func connect(dev *device.Device1) error {

	props := dev.Properties
	log.Info().Msg(fmt.Sprintf("Found device name=%s addr=%s rssi=%d", props.Name, props.Address, props.RSSI))

	if props.Connected {
		return nil
	}

	if !props.Paired {
		log.Info().Msg("Pairing device")
		err := dev.Pair()
		if err != nil {
			return fmt.Errorf("Pair failed: %s", err)
		}
	}

	log.Info().Msg("Connecting device")
	err := dev.Connect()
	if err != nil {
		return fmt.Errorf("Connect failed: %s", err)
	}

	log.Debug().Msg("Device connected")
	return nil
}
