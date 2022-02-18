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

FROM alpine:3.14.3@sha256:5e604d3358ab7b6b734402ce2e19ddd822a354dc14843f34d36c603521dbb4f9

LABEL maintainer "vollschwitz@gmx.net"

ARG binaries

#
# check for available Skopeo here:
#	https://pkgs.alpinelinux.org/packages?name=skopeo&branch=v3.15&arch=x86_64
#
RUN apk --update add --no-cache skopeo=1.3.1-r1 ca-certificates

COPY ${binaries}/dregsy /usr/local/bin

CMD ["dregsy", "-config=config.yaml"]
