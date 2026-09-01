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

FROM docker.io/golang:1.24.13-alpine3.22@sha256:3641e0d9b931dc4f2f185dcd669c4679670e9277c8166a838ddb98a2d4389cb5 AS skopeo

ARG SKOPEO_VERSION

RUN apk add --no-cache \
        git \
        make \
        gcc \
        musl-dev \
        pkgconfig \
        gpgme-dev \
        libassuan-dev \
        btrfs-progs-dev \
        libseccomp-dev \
        ostree-dev \
        containers-common \
        go-md2man

RUN git clone --depth 1 --branch ${SKOPEO_VERSION} \
        https://github.com/containers/skopeo.git /tmp/skopeo

WORKDIR /tmp/skopeo

RUN make bin/skopeo

FROM alpine:3.22

RUN apk add --no-cache \
        ca-certificates \
        gpgme \
        tzdata \
        openssl

COPY --from=skopeo /tmp/skopeo/bin/skopeo /usr/local/bin/skopeo

ARG binaries=_build/bin

COPY ${binaries}/dregsy /usr/local/bin/dregsy

ENTRYPOINT ["/usr/local/bin/dregsy"]
