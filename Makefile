REPO=dregsy

.PHONY: vendor build docker

build: vendor
	docker run --rm \
		-v $$(pwd):/go/src/github.com/xelalexv/$(REPO) \
		-w /go/src/github.com/xelalexv/$(REPO) \
		-e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 \
		golang:1.10 go build -v -a -tags netgo -installsuffix netgo \
		-ldflags '-w' -o dregsy ./cmd/dregsy/

docker: vendor
	docker build -t xelalex/$(REPO) -f Dockerfile .
	docker image prune --force --filter label=stage=intermediate

vendor:
	go get github.com/kardianos/govendor
	govendor sync
