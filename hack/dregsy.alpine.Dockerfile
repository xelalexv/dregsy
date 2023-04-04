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

# Skopeo build, taken from https://github.com/bdwyertech/docker-skopeo
#
FROM docker.io/golang:1.20.2-alpine3.17@sha256:bba3c7269b3d4aa875230617947481439b92cab8853b59ff8709fb047587c2eb as skopeo

ARG SKOPEO_VERSION

WORKDIR /go/src/github.com/containers/skopeo

RUN apk add --no-cache --virtual .build-deps \
        git build-base btrfs-progs-dev gpgme-dev linux-headers lvm2-dev \
    && git clone --single-branch --branch "${SKOPEO_VERSION}" \
        https://github.com/containers/skopeo.git . \
    && go build -ldflags="-s -w" -o bin/skopeo ./cmd/skopeo \
    && apk del .build-deps


# dregsy image
#
FROM docker.io/alpine:3.17.3@sha256:b6ca290b6b4cdcca5b3db3ffa338ee0285c11744b4a6abaa9627746ee3291d8d

LABEL maintainer "vollschwitz@gmx.net"

ARG binaries

RUN apk update && apk upgrade && apk --update add --no-cache \
    ca-certificates \
    device-mapper-libs \
    gpgme

COPY --from=skopeo /go/src/github.com/containers/skopeo/bin/skopeo /usr/bin
COPY ${binaries}/dregsy /usr/local/bin

CMD ["dregsy", "-config=config.yaml"]
