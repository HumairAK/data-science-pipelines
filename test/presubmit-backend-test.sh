#!/bin/bash
#
# Copyright 2020 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Fail the entire script when any command fails.
set -ex

# Add installed go binaries to PATH.
export PATH="${PATH}:$(go env GOPATH)/bin"

# 1. Reference local proto packages
go mod edit -replace github.com/kubeflow/pipelines/kubernetes_platform=./kubernetes_platform
go mod edit -replace github.com/kubeflow/pipelines/api=./api

# 2. Check go modules are tidy
# Reference: https://github.com/golang/go/issues/27005#issuecomment-564892876
go mod download
go mod tidy

# 1. Run tests in the backend directory
go test -v -cover ./backend/...
