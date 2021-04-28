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

FROM docker.io/ubuntu:20.04@sha256:c65d2b75a62135c95e2c595822af9b6f6cf0f32c11bcd4a38368d7b7c36b66f5

LABEL maintainer "vollschwitz@gmx.net"

ARG binaries

ENV DEBIAN_FRONTEND=noninteractive
ENV APT_KEY_DONT_WARN_ON_DANGEROUS_USAGE=yes

#
# FIXME: The kubic project only maintains one Skopeo version, i.e. as soon as
#        they release a new one, the older one gets removed. As a result, this
#        Docker build will fail. This makes it hard to stay at a particular
#        Skopeo version so that we stay in sync with the version in the Alpine
#        based image.
#
#        --> Find a package source without amnesia, or switch to newer Ubuntu
#            base image, which would have Skopeo in its repos already.
#

RUN apt-get update && \
    apt-get upgrade -y --fix-missing && \
    apt-get install -y --no-install-recommends --fix-missing \
        ca-certificates \
        apt-utils \
        gpg \
        curl && \
    echo 'deb http://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/xUbuntu_20.04/ /' \
        > /etc/apt/sources.list.d/devel:kubic:libcontainers:stable.list && \
    curl -fsSL https://download.opensuse.org/repositories/devel:kubic:libcontainers:stable/xUbuntu_20.04/Release.key \
        | gpg --dearmor > /etc/apt/trusted.gpg.d/devel_kubic_libcontainers_stable.gpg && \
    apt-get update && \
    apt-get install -y --no-install-recommends --fix-missing \
        skopeo=100:1.2.3-1 && \
    apt-get clean -y && \
    rm -rf \
        /var/cache/debconf/* \
        /var/lib/apt/lists/* \
        /var/log/* \
        /tmp/* \
        /var/tmp/*

COPY ${binaries}/dregsy /usr/local/bin

CMD ["dregsy", "-config=config.yaml"]
