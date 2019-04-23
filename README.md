# MDroid
RESTful API written in GOLANG for providing and logging vehicle session data. Essentially a backend to interfaces like [PyBus](https://github.com/MrDoctorKovacic/pyBus).

## Requirements
* Go v1.11 at a minimum ([Raspberry Pi Install](https://gist.github.com/kbeflo/9d981573aad107da6fa7ac0603259b3b)) 

## Installation 

Having [InfluxDB & the rest of the TICK stack](https://www.influxdata.com/blog/running-the-tick-stack-on-a-raspberry-pi/) is recommended, although this will run fine without them.

```go get github.com/MrDoctorKovacic/MDroid-Core/``` 

## Usage

```MDroid-Core --settings-file ./settings.json``` 

## Options 

The settings file is a simple json formatted document for program init and generic settings that should persist through multiple runs. If the settings file argument is omitted, or the file is not found, one will be created.

Any and all options can be omitted to disable their specific functionality. In the case of missing database options, the program will still keep track of the current session but it won't log the time series data. 

```
// COMMENTS ARE NOT VALID JSON, AN UNCOMMENTED VERSION OF THE BELOW IS PROVIDED IN THE REPO
// Core Configuration
{
	"config": {
		"DatabaseHost": "http://localhost:8086", // Influx DB with port
		"DatabaseName": "vehicle", // Influx DB name to aggregate data under
		"BluetoothAddress": "", // This is NOT the host's BT Addr, rather the default media device
		"PingHost": "", // Mostly proprietary, safe to ignore
		"SettingsFile": "./session.json" // 
	}
}
``` 

## Settings File

Inferred settings will appear here. Allows for posting generic setting fields and values, which can be retrieved later by others in the network. This MUST be a 2D array, matching the (settings[Component][Field] = Value) style. Inner casing is however user defines, but I generally prefer upper. Publish / Subscribe for settings is a planned feature.

Here's an example of a generated JSON settings structure:

```
{
	"settings": {
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
}
```