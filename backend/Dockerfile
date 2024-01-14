# Copyright 2021-2022 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# 1. Build api server application
FROM golang:1.20.4-buster as builder
RUN apt-get update && apt-get install -y cmake clang musl-dev openssl
WORKDIR /go/src/github.com/kubeflow/pipelines
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN GO111MODULE=on go build -o /bin/apiserver backend/src/apiserver/*.go
WORKDIR /
COPY backend/src/apiserver/config/sample_config.json /samples/

# 3. Start api web server
FROM debian:stable

ARG COMMIT_SHA=unknown
ENV COMMIT_SHA=${COMMIT_SHA}
ARG TAG_NAME=unknown
ENV TAG_NAME=${TAG_NAME}

WORKDIR /bin

COPY backend/src/apiserver/config/ /config
COPY --from=builder /bin/apiserver /bin/apiserver
COPY --from=builder /samples/ /samples/
RUN chmod +x /bin/apiserver
RUN apt-get update && apt-get install -y ca-certificates wget
# Expose apiserver port
EXPOSE 8888

# Start the apiserver
CMD /bin/apiserver --config=/config --sampleconfig=/config/sample_config.json -logtostderr=true
