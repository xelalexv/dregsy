#!/bin/bash
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o dregsy ./cmd/dregsy/
docker build -t xelalex/dregsy -f Dockerfile .
rm ./dregsy
