#
# Copyright 2021 Alexander Vollschwitz
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

# Skopeo build
#
FROM golang:1.18.3@sha256:5417b4917fa7ed3ad2678a3ce6378a00c95bfd430c2ffa39936fce55130b5f2c as skopeo

ARG SKOPEO_VERSION

WORKDIR /go/src/github.com/containers/skopeo

RUN apt-get update \
    && apt-get install -y --no-install-recommends --fix-missing \
        git libbtrfs-dev libgpgme-dev liblvm2-dev \
    && git clone --single-branch --branch "${SKOPEO_VERSION}" \
        https://github.com/containers/skopeo.git . \
    && go build -ldflags="-s -w" -o bin/skopeo ./cmd/skopeo


# dregsy image
#
FROM docker.io/ubuntu:22.04@sha256:aa6c2c047467afc828e77e306041b7fa4a65734fe3449a54aa9c280822b0d87d

LABEL maintainer "vollschwitz@gmx.net"

ARG binaries

ENV DEBIAN_FRONTEND=noninteractive
ENV APT_KEY_DONT_WARN_ON_DANGEROUS_USAGE=yes

#
# check for available Skopeo here:
#   https://packages.ubuntu.com/search?keywords=skopeo&searchon=names&suite=jammy&section=all
#
RUN apt-get update && \
    apt-get upgrade -y --fix-missing && \
    apt-get install -y --no-install-recommends --fix-missing \
        ca-certificates \
        apt-utils \
        gpg \
        curl \
        libgpgme11 \
        libdevmapper1.02.1 && \
    apt-get clean -y && \
    rm -rf \
        /var/cache/debconf/* \
        /var/lib/apt/lists/* \
        /var/log/* \
        /tmp/* \
        /var/tmp/*

COPY --from=skopeo /go/src/github.com/containers/skopeo/bin/skopeo /usr/bin
COPY ${binaries}/dregsy /usr/local/bin

CMD ["dregsy", "-config=config.yaml"]
