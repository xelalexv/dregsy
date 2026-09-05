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

FROM docker.io/golang:1.24.13@sha256:d2d2bc1c84f7e60d7d2438a3836ae7d0c847f4888464e7ec9ba3a1339a1ee804 AS skopeo

ARG SKOPEO_VERSION

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        git \
        make \
        gcc \
        pkg-config \
        libgpgme-dev \
        libassuan-dev \
        libbtrfs-dev \
        libdevmapper-dev \
        libseccomp-dev \
        libostree-dev \
        ca-certificates \
        libgpgme11 \
        go-md2man && \
    rm -rf /var/lib/apt/lists/*

RUN git clone --depth 1 --branch ${SKOPEO_VERSION} \
        https://github.com/containers/skopeo.git /go/src/github.com/containers/skopeo

WORKDIR /go/src/github.com/containers/skopeo

RUN make bin/skopeo

FROM docker.io/golang:1.24.13@sha256:d2d2bc1c84f7e60d7d2438a3836ae7d0c847f4888464e7ec9ba3a1339a1ee804 AS dregsy

WORKDIR /src

COPY . .

ARG DREGSY_VERSION

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -X main.DregsyVersion=${DREGSY_VERSION}" \
    -o /out/dregsy \
    ./cmd/dregsy

FROM ubuntu:24.04

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        libgpgme11 \
        tzdata \
        openssl && \
    apt-get clean && \
    rm -rf \
        /var/lib/apt/lists/* \
        /var/cache/apt/* \
        /tmp/* \
        /var/tmp/*

COPY --from=skopeo /go/src/github.com/containers/skopeo/bin/skopeo /usr/local/bin/skopeo
COPY --from=dregsy /out/dregsy /usr/local/bin/dregsy

ENTRYPOINT ["/usr/local/bin/dregsy"]
