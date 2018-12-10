REPO=dregsy
SKOPEO_DIR = ./third_party/skopeo

.PHONY: vendor build docker skopeo

build: vendor
	docker run --rm --user $(shell id -u):$(shell id -g) \
		-v $$(pwd):/go/src/github.com/xelalexv/$(REPO) \
		-w /go/src/github.com/xelalexv/$(REPO) \
		-e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 \
		golang:1.10 go build -v -a -tags netgo -installsuffix netgo \
		-ldflags '-w' -o dregsy ./cmd/dregsy/

docker: vendor skopeo
	docker build -t xelalex/$(REPO) -f Dockerfile .
	docker image prune --force --filter label=stage=intermediate

vendor:
	docker run --rm \
		-v $$(pwd):/go/src/github.com/xelalexv/$(REPO) \
		-w /go/src/github.com/xelalexv/$(REPO) \
		golang:1.10 go get github.com/kardianos/govendor && govendor sync

skopeo:
	git submodule update --init
	$(MAKE) -C $(SKOPEO_DIR) binary-static DISABLE_CGO=1
