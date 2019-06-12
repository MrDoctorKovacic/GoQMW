# MDroid-Core

[![Build Status](https://travis-ci.org/MrDoctorKovacic/MDroid-Core.svg?branch=master)](https://travis-ci.org/MrDoctorKovacic/MDroid-Core) [![Go Report Card](https://goreportcard.com/badge/github.com/MrDoctorKovacic/MDroid-Core)](https://goreportcard.com/report/github.com/MrDoctorKovacic/MDroid-Core)

REST API for an edge device, which acts as middleman for vehicle data. 

Essentially a backend to my own interfaces like [PyBus](https://github.com/MrDoctorKovacic/pyBus) or other inputs (think GPS, CAN, etc). This aggregates data from various sources to be retrieved by other programs or logged for later analysis. Also used as a hub to delegate specific actions to node devices.

## Benefits
* Incoming data is logged to [InfluxDB](https://www.influxdata.com/): a performant time series Database.
* Pipelines vehicle information to one location that can be reliably queried.
* Global state answers questions like "When should the running lights be on?" to anyone on the network that asks.
* Goes both ways! Allows for sending remote commands over BMW I-Bus (with [PyBus](https://github.com/MrDoctorKovacic/pyBus)). CAN-Bus writes are planned.
* Status reporting keeps an eye on a network of temperamental devices, helps squash bad behavior.
* It's written in Go and relatively quick. It can (and does) run on OpenWRT ARM routers using the MUSL compiler. Try it, [the MUSL binary is cross-compiled and included.](https://github.com/MrDoctorKovacic/MDroid-Core/blob/master/bin/MDroid-Core-MUSL)

## Requirements
* Go v1.11 at a minimum ([Raspberry Pi Install](https://gist.github.com/kbeflo/9d981573aad107da6fa7ac0603259b3b)) 

## Installation 

Having [InfluxDB & the rest of the TICK stack](https://www.influxdata.com/blog/running-the-tick-stack-on-a-raspberry-pi/) is recommended, although a neutered version will run fine without them.

```go get github.com/MrDoctorKovacic/MDroid-Core/``` 

## Usage

```MDroid-Core --settings-file ./settings.json``` 

## Configuration 

The `settings.json` file is a simple JSON document for program settings that should persist through each load. The ones under the header `CONFIG` are suggested for the program to function properly. The program provides endpoints for user-defined settings to be POST-ed at will. 

This allows for setting generic fields and values, which can be retrieved later. Some notes:

* This MUST be a 2D array, matching the (settings[Component][Field] = Value) style.
* If the settings file is omitted or missing, one will be created.
* `CONFIG` options can be omitted to disable their specific functionality.

Here's a commented example with suggested settings:

```
// COMMENTS ARE NOT VALID JSON, AN UNCOMMENTED VERSION OF THE BELOW IS PROVIDED IN THE REPO
{
	// Core Configuration
	"CONFIG": {
		"CORE_DATABASE_HOST": "http://localhost:8086", // Influx DB with port
		"CORE_DATABASE_NAME": "vehicle", // Influx DB name to aggregate data under
		"BLUETOOTH_ADDRESS": "", // This is NOT the host's BT Addr, rather the default media device
		"CORE_PING_HOST": "", // Mostly proprietary, safe to ignore
	}
	
	// Examples of user defined settings, these won't do anything and only store values
	// for other programs to retrieve
	"ANGEL_EYES": {
		"POWER": 1,
		"TURN_OFF_WHEN": "NIGHT"
	},
	"BACKUP_CAMERA": {
		...
	}
	...
}
``` 

### What's the difference between settings and session values?

All incoming data typically falls into one of these two categories. They are made distinct with entirely different endpoints and processing. But together they make up the bulk of query-able vehicle information. 

Generally, **Settings** will define values that should persist through reboots of the program and device (since these are saved to disk frequently). Examples:
* Wait time after power loss to shutdown
* Vehicle lighting mode
* Meta program settings

**Session** values are better suited to in-the-moment last-known data. This is because the session is never explicitly written to disk, and performs better with a high volume of data. It also wouldn't make sense to save (and later load) my car's speed from last week as a definitive value. Examples:
* Speed
* RPM
* GPS fixes
* License plate sightings

In terms of logging, Session values are the more interesting to see change over time.