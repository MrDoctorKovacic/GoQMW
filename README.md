# MDroid

[![Go Report Card](https://goreportcard.com/report/github.com/MrDoctorKovacic/MDroid-Core)](https://goreportcard.com/report/github.com/MrDoctorKovacic/MDroid-Core)

RESTful API written in GOLANG for providing and logging vehicle session data. Essentially a backend to interfaces like [PyBus](https://github.com/MrDoctorKovacic/pyBus).

## Requirements
* Go v1.11 at a minimum ([Raspberry Pi Install](https://gist.github.com/kbeflo/9d981573aad107da6fa7ac0603259b3b)) 

## Installation 

Having [InfluxDB & the rest of the TICK stack](https://www.influxdata.com/blog/running-the-tick-stack-on-a-raspberry-pi/) is recommended, although this will run fine without them.

```go get github.com/MrDoctorKovacic/MDroid-Core/``` 

## Usage

```MDroid-Core --settings-file ./settings.json``` 

## Configuration 

The config file is a simple json formatted document for program initialization. Here's a commented example with available settings:

```
// COMMENTS ARE NOT VALID JSON, AN UNCOMMENTED VERSION OF THE BELOW IS PROVIDED IN THE REPO
// Core Configuration
{
	"DatabaseHost": "http://localhost:8086", // Influx DB with port
	"DatabaseName": "vehicle", // Influx DB name to aggregate data under
	"BluetoothAddress": "", // This is NOT the host's BT Addr, rather the default media device
	"PingHost": "", // Mostly proprietary, safe to ignore
	"SettingsFile": "./settings.json" // Where persistient settings should be stored
}
``` 

## Generated Settings File

Inferred settings will appear here. Allows for posting generic setting fields and values, which can be retrieved later by others in the network. This MUST be a 2D array, matching the (settings[Component][Field] = Value) style.

If the settings file argument is omitted, or the file is not found, one will be created. Any and all options can be omitted to disable their specific functionality. In the case of missing database options, the program will still keep track of the current session but it won't log the time series data. 

Publish / Subscribe for settings is a planned feature.

Here's an example of a generated JSON settings structure:

```
{
	// Example: set angel eyes to be turned on. As of now, the controller should be listening on this value
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