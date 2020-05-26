#
# Copyright 2020 Alexander Vollschwitz
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

.DEFAULT_GOAL := help
SHELL = /bin/bash

REPO=dregsy
DREGSY_VERSION=$$(git describe --always --tag --dirty)

BUILD_OUTPUT=build
BINARIES=$(BUILD_OUTPUT)/bin
ISOLATED_PKG=$(BUILD_OUTPUT)/pkg
ISOLATED_CACHE=$(BUILD_OUTPUT)/cache

GO_IMAGE=golang:1.13.6-buster@sha256:f6cefbdd25f9a66ec7dcef1ee5deb417882b9db9629a724af8a332fe54e3f7b3

##
# You can set the following environment variables when calling make:
#
#	${ITL}VERBOSE=y${NRM}	get detailed output
#
#	${ITL}ISOLATED=y${NRM}	when using this with a build target, the build will be isolated
#			in the sense that local caches such as ${DIM}\${GOPATH}/pkg${NRM} and ${DIM}~/.cache${NRM}
#			will not be mounted into the build container. Instead, according
#			folders underneath ${DIM}./build${NRM} are used. These folders are removed when
#			running ${DIM}make clean${NRM}. That way you can force a full build, where all
#			dependencies are retrieved & built inside the container.
#

VERBOSE ?=
ifeq ($(VERBOSE),y)
    $(warning ***** starting Makefile for goal(s) "$(MAKECMDGOALS)")
    $(warning ***** $(shell date))
    MAKEFLAGS += --trace
else
    MAKEFLAGS += -s
endif

ifeq ($(MAKECMDGOALS),release)
	ISOLATED=y
endif

ISOLATED ?=
ifeq ($(ISOLATED),y)
    CACHE_VOLS=-v $$(pwd)/$(ISOLATED_PKG):/go/pkg -v $$(pwd)/$(ISOLATED_CACHE):/.cache
else
    CACHE_VOLS=-v $(GOPATH)/pkg:/go/pkg -v /home/$(USER)/.cache:/.cache
endif

export

#
#

.PHONY: help
help:
#	show this help
#
	$(call utils, synopsis) | more


.PHONY: release
release: clean dregsy image
#	clean, do an isolated build, and create container image
#


.PHONY: dregsy
dregsy:
#	build the ${ITL}dregsy${NRM} binary
#
	mkdir -p $(BINARIES) $(ISOLATED_PKG) $(ISOLATED_CACHE)
	docker run --rm --user $(shell id -u):$(shell id -g) \
        -v $(shell pwd)/$(BINARIES):/go/bin $(CACHE_VOLS) \
		-v $(shell pwd):/go/src/$(REPO) -w /go/src/$(REPO) \
		-e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 \
		$(GO_IMAGE) go build -v -a -tags netgo -installsuffix netgo \
		-ldflags "-w -X main.DregsyVersion=$(DREGSY_VERSION)" \
		-o $(BINARIES)/dregsy ./cmd/dregsy/


.PHONY: image
image:
#	build the ${ITL}dregsy${NRM} container image; assumes binary was built
#
	docker build -t xelalex/$(REPO) -f Dockerfile \
		--build-arg binaries=$(BINARIES) .


.PHONY: clean
clean:
#	remove all build artifacts, including isolation caches
#
	[[ ! -d $(BUILD_OUTPUT) ]] || chmod -R u+w $(BUILD_OUTPUT)
	rm -rf $(BUILD_OUTPUT)


#
# helper functions
#
utils = ./hack/devenvutil $(1)
