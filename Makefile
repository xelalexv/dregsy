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

BUILD_OUTPUT=_build
BINARIES=$(BUILD_OUTPUT)/bin
ISOLATED_PKG=$(BUILD_OUTPUT)/pkg
ISOLATED_CACHE=$(BUILD_OUTPUT)/cache

GO_IMAGE=golang:1.13.6-buster@sha256:f6cefbdd25f9a66ec7dcef1ee5deb417882b9db9629a724af8a332fe54e3f7b3

## makerc
# You need to set the following parameters in configuration file ${DIM}.makerc${NRM}, with every line
# containing a parameter in the form ${ITL}key = value${NRM}:
#
#	${ITL}DREGSY_TEST_ECR_REGISTRY${NRM}	the ECR instance to use
#	${ITL}DREGSY_TEST_ECR_REPO${NRM} 		the repo to use within the ECR instance;
#					defaults to ${DIM}dregsy/test${NRM}
#
#	${ITL}AWS_ACCESS_KEY_ID${NRM}	credentials for AWS account in which ECR instance for testing
#	${ITL}AWS_SECRET_ACCESS_KEY${NRM}	is located; the user associated with these credentials needs to
#				have sufficient IAM permissions for creating an ECR instance,
#				pulling & pushing from/to it, and deleting it
#
#	If any of the above settings without a default is missing, ECR tests are skipped!
#
-include .makerc

## env
# You can set the following environment variables when calling make:
#
#	${ITL}VERBOSE=y${NRM}	get detailed output
#
#	${ITL}ISOLATED=y${NRM}	when using this with a build or test target, the build/test will be isolated
#			in the sense that local caches such as ${DIM}\${GOPATH}/pkg${NRM} and ${DIM}~/.cache${NRM} will
#			not be mounted into the container. Instead, according folders underneath
#			the configured build folder are used. These folders are removed when
#			running ${DIM}make clean${NRM}. That way you can force a clean build/test, where all
#			dependencies are retrieved & built inside the container.
#

VERBOSE ?=
ifeq ($(VERBOSE),y)
    $(warning ***** starting Makefile for goal(s) "$(MAKECMDGOALS)")
    $(warning ***** $(shell date))
    MAKEFLAGS += --trace
    TEST_OPTS = -v
else
    MAKEFLAGS += -s
    TEST_OPTS =
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
release: clean dregsy imgdregsy imgtests tests
#	clean, do an isolated build, create container images, and test
#


.PHONY: dregsy
dregsy: prep
#	build the ${ITL}dregsy${NRM} binary
#
	docker run --rm --user $(shell id -u):$(shell id -g) \
        -v $(shell pwd)/$(BINARIES):/go/bin $(CACHE_VOLS) \
		-v $(shell pwd):/go/src/$(REPO) -w /go/src/$(REPO) \
		-e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 \
		$(GO_IMAGE) go build -v -tags netgo -installsuffix netgo \
		-ldflags "-w -X main.DregsyVersion=$(DREGSY_VERSION)" \
		-o $(BINARIES)/dregsy ./cmd/dregsy/


.PHONY: imgdregsy
imgdregsy:
#	build the ${ITL}dregsy${NRM} container image; assumes binary was built
#
	docker build -t xelalex/$(REPO) -f ./hack/dregsy.Dockerfile \
		--build-arg binaries=$(BINARIES) .


.PHONY: imgtests
imgtests:
#	build the container image for running tests; assumes ${ITL}dregsy${NRM} image was built
#
	docker build -t xelalex/$(REPO)-tests -f ./hack/tests.Dockerfile .


.PHONY: tests
tests: prep
#	run tests; assumes tests image was built and local ${ITL}Docker${NRM} registry running
#	on localhost:5000 (start with ${DIM}make registryup${NRM});
#
ifeq (,$(wildcard .makerc))
	$(warning ***** Missing .makerc! Some tests may be skipped or fail!)
endif
	@echo -e "\ntests:"
	docker run --privileged --rm  \
		-v $(shell pwd):/go/src/$(REPO) -w /go/src/$(REPO) \
        -v $(shell pwd)/$(BINARIES):/go/bin $(CACHE_VOLS) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 \
		-e LOG_LEVEL=debug -e LOG_FORMAT=text -e LOG_FORCE_COLORS=true \
		--env-file <(sed -E 's/\ +=\ +/=/g' .makerc) \
		xelalex/$(REPO)-tests sh -c "\
			go test $(TEST_OPTS) \
				-coverpkg=./... -coverprofile=$(BUILD_OUTPUT)/coverage.out \
				-covermode=count ./... && \
			go tool cover -html=$(BUILD_OUTPUT)/coverage.out \
				-o $(BUILD_OUTPUT)/coverage.html"
	@echo -e "\ncoverage report is in $(BUILD_OUTPUT)/coverage.html\n"


.PHONY: registryup
registryup:
#	start local ${ITL}Docker${NRM} registry for running tests
#
	docker run -d --rm -p 5000:5000 --name dregsy-test-registry registry:2


.PHONY: registrydown
registrydown:
#	stop local ${ITL}Docker${NRM} registry
#
	docker stop dregsy-test-registry


.PHONY: clean
clean:
#	remove all build artifacts, including isolation caches
#
	[ ! -d $(BUILD_OUTPUT) ] || chmod -R u+w $(BUILD_OUTPUT)
	rm -rf $(BUILD_OUTPUT)/*


.PHONY: prep
prep:
#	prepare required directories
#
	mkdir -p $(BUILD_OUTPUT) $(BINARIES) $(ISOLATED_PKG) $(ISOLATED_CACHE)


#
# helper functions
#
utils = ./hack/devenvutil $(1)
