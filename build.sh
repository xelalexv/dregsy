#!/bin/bash
set -e
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o dregsy ./cmd/dregsy/
if [ "$1" == "-d" ]; then
	docker build -t xelalex/dregsy -f Dockerfile .
	rm ./dregsy
fi
