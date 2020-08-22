# MDroid-Core

[![Build Status](https://travis-ci.org/qcasey/MDroid-Core.svg?branch=master)](https://travis-ci.org/qcasey/MDroid-Core) [![Go Report Card](https://goreportcard.com/badge/github.com/qcasey/MDroid-Core)](https://goreportcard.com/report/github.com/qcasey/MDroid-Core)

![Controls](https://quinncasey.com/wp-content/uploads/2019/10/Arrays-Web-1.jpg "Screenshot 1")
[Control App](https://github.com/qcasey/MDroid-Control)

![Watch Intgration](https://quinncasey.com/wp-content/uploads/2019/09/maxresdefault.jpg)

[Galaxy Watch Integration](https://quinncasey.com/unlocking-vehicle-with-mdroid-core-from-smartwatch/)

## Motivation

I wanted a hub to ingest different kinds of data from sources on my car, as well as store this data and make it available for other programs. Some sources are from my interfaces to the stock buses like [PyBus](https://github.com/qcasey/pyBus) or [CAN](https://github.com/qcasey/MDroid-CAN), others are inputs like [GPS](https://github.com/qcasey/MDroid-GPS) or [Drok UART](https://github.com/qcasey/MDroid-Drok).

 The board it's riding has an always-on LTE connection, giving me real time updates and control. Inspired by [Tesla's app implementation](https://www.tesla.com/support/tesla-app).

## Benefits

* Incoming data is stored in [InfluxDB](https://www.influxdata.com/): a performant time series Database.
* Pipelines data to one location that can be reliably queried.
* Stores persistent settings for other machines on the network.
* Can lower windows, open trunk, turn on hazards remotely, etc by mapping queries to the [BMW K-Bus](https://github.com/qcasey/pyBus)).
* It's written in Go, runs on OpenWRT ARM boards in the MUSL compiler. Try it, [the MUSL bin is cross-compiled.](https://github.com/qcasey/MDroid-Core/blob/master/bin/MDroid-Core-MUSL)

## Requirements

* Golang 1.13+ ([Raspberry Pi Install](https://gist.github.com/kbeflo/9d981573aad107da6fa7ac0603259b3b))

## Installation

Having [InfluxDB & the rest of the TICK stack](https://www.influxdata.com/blog/running-the-tick-stack-on-a-raspberry-pi/) is recommended, although a neutered version will run fine without it.

```go get github.com/qcasey/MDroid-Core/```

## Usage

```MDroid-Core --settings-file ./settings.json```

### The difference between settings and session values

Generally, **Settings** are persistent and saved to disk frequently. **Session** values are not.

**Example Setting Values**
* Wait time after power loss to shutdown
* Vehicle lighting mode
* Meta program settings

**Example Session Values**
* Speed
* RPM
* GPS fixes
* License plate sightings

Naturally, Session values are the more interesting to see change over time.
