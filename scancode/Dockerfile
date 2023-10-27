# Copyright 2023 The Authors (see AUTHORS file)
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

FROM python:3.10

ENV SCANCODE_RELEASE=32.0.8

RUN apt-get update && apt-get install -y bzip2 xz-utils zlib1g libxml2-dev libxslt1-dev

ADD "https://github.com/nexB/scancode-toolkit/archive/refs/tags/v${SCANCODE_RELEASE}.tar.gz" .

RUN mkdir scancode-toolkit && tar xzvf v${SCANCODE_RELEASE}.tar.gz -C scancode-toolkit --strip-components=1

WORKDIR /scancode-toolkit

RUN ./scancode --help

ENV PATH=$HOME/scancode-toolkit:$PATH

RUN pip3 install pyyaml

COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
