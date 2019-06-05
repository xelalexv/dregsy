REPO=dregsy
SKOPEO_DIR=./third_party/skopeo
DREGSY_VERSION=$$(git describe --always --tag --dirty)

.PHONY: vendor build docker skopeo

build: vendor
	docker run --rm --user $(shell id -u):$(shell id -g) \
		-v $$(pwd):/go/src/github.com/xelalexv/$(REPO) \
		-w /go/src/github.com/xelalexv/$(REPO) \
		-e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 \
		-e GOCACHE=/go/src/github.com/xelalexv/$(REPO)/.cache \
		golang:1.10 go build -v -a -tags netgo -installsuffix netgo \
		-ldflags "-w -X main.DregsyVersion=$(DREGSY_VERSION)" \
		-o dregsy ./cmd/dregsy/

docker: vendor skopeo
	docker build -t xelalex/$(REPO) -f Dockerfile \
		--build-arg dregsy_version=$(DREGSY_VERSION) .
	docker image prune --force --filter label=stage=intermediate

vendor:
	docker run --rm \
		-v $$(pwd):/go/src/github.com/xelalexv/$(REPO) \
		-w /go/src/github.com/xelalexv/$(REPO) \
		golang:1.10 "bash -c go get github.com/kardianos/govendor && govendor sync"

skopeo:
	git submodule update --init
	# issue 7: patch Skopeo's build Dockerfile to use more recent Ubuntu
	sed -i 's/FROM ubuntu:17.10/FROM ubuntu:18.10/' $(SKOPEO_DIR)/Dockerfile.build
	$(MAKE) -C $(SKOPEO_DIR) binary-static DISABLE_CGO=1
	# issue 7: restore original Dockerfile
	cd $(SKOPEO_DIR); git checkout Dockerfile.build
