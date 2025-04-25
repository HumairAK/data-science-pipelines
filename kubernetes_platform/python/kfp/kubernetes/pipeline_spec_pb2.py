# Copyright 2025 The Kubeflow Authors
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

# This file is a bit of a hack / workaround to allow kubernetes_executor_config_pb2.py
# and non-proto generated code to leverage pipeline_spec proto message fields
# that is shared with pipeline spec proto. Symbols need to be manually assigned
# to enable intellisense.

import kfp.pipeline_spec.pipeline_spec_pb2 as _real_pb2

# Take all the names from _real_pb2 and inject them into the current module's namespace.
globals().update(vars(_real_pb2))

# Specify a symbol here to enable intellisense
# This will allow tools like ides to be able to
# do things like type hinting on this symbol
# it is not required for compilation.
TaskInputsSpec = _real_pb2.TaskInputsSpec
