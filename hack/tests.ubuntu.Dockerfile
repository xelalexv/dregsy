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

FROM xelalex/dregsy:latest-ubuntu

# install & configure Go
RUN apt-get update && \
    apt-get install -y --no-install-recommends --fix-missing \
        golang && \
    apt-get clean -y && \
    rm -rf \
        /var/cache/debconf/* \
        /var/lib/apt/lists/* \
        /var/log/* \
        /tmp/* \
        /var/tmp/*

ENV GOROOT /usr/lib/go
ENV GOPATH /go
ENV GOCACHE /.cache
ENV PATH /go/bin:${PATH}
RUN mkdir -p ${GOPATH}/src ${GOPATH}/bin ${GOPATH}/pkg ${GOCACHE}

# non-root user
ARG USER=go
RUN groupadd -o -g ${GROUP_ID:-1000} ${USER} && \
    useradd -l -o -u ${USER_ID:-1000} -g ${USER} ${USER} && \
    install -d -m 0755 -o ${USER} -g ${USER} /home/${USER}
ENV HOME /home/${USER}
USER ${USER}

WORKDIR ${GOPATH}

CMD ["go", "version"]
