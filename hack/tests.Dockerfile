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

FROM xelalex/dregsy:latest

# install & configure Go
RUN apk add --no-cache go
ENV GOROOT /usr/lib/go
ENV GOPATH /go
ENV GOCACHE /.cache
ENV PATH /go/bin:${PATH}
RUN mkdir -p ${GOPATH}/src ${GOPATH}/bin ${GOPATH}/pkg ${GOCACHE}

# non-root user
ARG USER=go
ENV HOME /home/${USER}
RUN apk add --update sudo
RUN adduser -D ${USER} \
	&& adduser ${USER} ping \
    && echo "${USER} ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/${USER} \
    && chmod 0440 /etc/sudoers.d/${USER}
USER ${USER}

WORKDIR ${GOPATH}

CMD ["go", "version"]
