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

# Build dregsy from source using the required Go toolchain.
FROM docker.io/golang:1.24.13 AS builder

WORKDIR /src

COPY . .

ARG DREGSY_VERSION

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -X main.DregsyVersion=${DREGSY_VERSION}" \
    -o /out/dregsy \
    ./cmd/dregsy


# Runtime image.
FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        jq \
        skopeo \
        tzdata && \
    apt-get clean && \
    rm -rf \
        /var/lib/apt/lists/* \
        /var/cache/apt/* \
        /var/log/* \
        /tmp/* \
        /var/tmp/*

COPY --from=builder /out/dregsy /usr/local/bin/dregsy

RUN chmod 0755 /usr/local/bin/dregsy

ENTRYPOINT ["/usr/local/bin/dregsy"]
