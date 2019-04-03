# GoQMW
RESTful API written in GOLANG for providing and logging vehicle session data. Essentially a backend to interfaces like [PyBus](https://github.com/MrDoctorKovacic/pyBus).

## Requirements
* Go v1.11 at a minimum ([Raspberry Pi Install](https://gist.github.com/kbeflo/9d981573aad107da6fa7ac0603259b3b)) 

## Installation 

Having [InfluxDB & the rest of the TICK stack](https://www.influxdata.com/blog/running-the-tick-stack-on-a-raspberry-pi/) is recommended, although this will run fine without them.

```go get github.com/MrDoctorKovacic/GoQMW/``` 

## Usage

Any and all the options can be omitted to disable their specific functionality. In the case of missing database options, the program will still keep track of the current session but it won't log the time series data. 

```GoQMW --db-host http://localhost:8086 --db-name vehicle --session-file ./session.json --bt-address [BT PLAYER MAC ADDR] --ping-host https://ping.quinncasey.com``` 

## Options 

* **db-host and db-name** control access to the Influx database for logging time series data. User auth is not supported. 
* **session-file** can save / recover from the most recent session, formatted in JSON. This is not a log, rather a freeze frame of the entire latest session. 
* **bt-address** is the MAC addr of a device that should be used by default. TODO: Allow for dynamic address change based on what is actually connected.
* **ping-host** should not really be used. All of my devices ping a master server on a regular interval to 'check in', measuring downtime and broken nodes. 