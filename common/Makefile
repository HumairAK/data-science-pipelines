# Copyright 2025 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Contact one of chensun, HumairAK, zijianjoy if this remote image needs an update.
PREBUILT_REMOTE_IMAGE=ghcr.io/kubeflow/kfp-api-generator:1.0

.PHONY: all
all: golang

.PHONY: clean
clean: clean-go

# Generate proto packages using a pre-built api-generator image.
.PHONY: golang
golang: *.proto
	docker run --interactive --rm \
		--user $$(id -u):$$(id -g) \
		--mount type=bind,source="$$(pwd)/..",target=/go/src/github.com/kubeflow/pipelines \
		$(PREBUILT_REMOTE_IMAGE)  \
		sh -c 'cd /go/src/github.com/kubeflow/pipelines/common && make generate'

# Delete all generated proto go packages.
.PHONY: clean-go
clean-go:
	rm -rf go
	rm -f google/rpc/status.proto

##########################
# The following are IMPLEMENTATION DETAILS.
##########################

# Generates proto packages locally, this should only be called:
# * during development
# * inside the prebuilt docker container
.PHONY: generate
generate: go_commonspec

go_commonspec: common.proto
	mkdir -p go/commonspec
	protoc -I=. \
		--go_out=go/commonspec \
		--go_opt=paths=source_relative \
		common.proto

# Fetch dependency proto
google/rpc/status.proto:
	mkdir -p google/rpc
	wget -O google/rpc/status.proto https://raw.githubusercontent.com/googleapis/googleapis/047d3a8ac7f75383855df0166144f891d7af08d9/google/rpc/status.proto

# protoc-gen-go is already installed in api-generator image
.PHONY: protoc-gen-go
protoc-gen-go:
	go install google.golang.org/protobuf/cmd/protoc-gen-go
