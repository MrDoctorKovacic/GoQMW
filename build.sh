#!/bin/bash

# MAC MUSL BUILD
#env CC=arm-linux-musleabihf-gcc GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=1 go build -o MDroid-Core-MUSL ./

# LINUX MUSL BUILD
env CC=arm-linux-musleabihf-gcc GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=1 go build -o ./bin/MDroid-Core-MUSL ./

# REGULAR GO BUILD
go build -o ./bin/MDroid-Core
