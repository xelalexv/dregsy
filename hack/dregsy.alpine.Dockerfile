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

FROM alpine:3.14.0@sha256:1775bebec23e1f3ce486989bfc9ff3c4e951690df84aa9f926497d82f2ffca9d

LABEL maintainer "vollschwitz@gmx.net"

ARG binaries

RUN apk --update add --no-cache skopeo=1.3.1-r0 ca-certificates

COPY ${binaries}/dregsy /usr/local/bin

CMD ["dregsy", "-config=config.yaml"]
