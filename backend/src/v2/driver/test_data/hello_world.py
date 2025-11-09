# Copyright 2021 The Kubeflow Authors
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
import functools
import os

from kfp import dsl
from kfp import compiler

base_image="quay.io/opendatahub/ds-pipelines-ci-executor-image:v1.0"
dsl.component = functools.partial(dsl.component, base_image=base_image)

@dsl.component()
def hello_world(text: str) -> str:
    print(text)
    return text


@dsl.pipeline(name='hello-world', description='A simple intro pipeline')
def pipeline_hello_world(text: str = 'hi there'):
    consume_task = hello_world(
        text=text)  # Passing pipeline parameter as argument to consumer op


if __name__ == '__main__':
    from kfp import compiler

    compiler.Compiler().compile(
        pipeline_func=pipeline_hello_world,
        package_path=__file__+".yaml"
    )
