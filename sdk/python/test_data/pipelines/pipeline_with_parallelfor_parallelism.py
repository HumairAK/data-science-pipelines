# Copyright 2022 The Kubeflow Authors
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

import os
import tempfile
from typing import Dict, List

from kfp import compiler
from kfp import components
from kfp import dsl
from kfp.dsl import component


@component
def print_text(msg: str):
    print(msg)


@component
def print_int(x: int):
    print(x)


@component
def list_dict_maker_0() -> List[Dict[str, int]]:
    """Enforces strict type checking - returns a list of dictionaries 
    where keys are strings and values are integers. For testing type 
    handling during compilation."""
    return [{'a': 1, 'b': 2}, {'a': 2, 'b': 3}, {'a': 3, 'b': 4}]


@component
def list_dict_maker_1() -> List[Dict]:
    """Utilizes generic dictionary typing (no enforcement of specific key or
    value types).

    Tests flexibility in type handling.
    """
    return [{'a': 1, 'b': 2}, {'a': 2, 'b': 3}, {'a': 3, 'b': 4}]


@component
def list_dict_maker_2() -> List[dict]:
    """Returns a list of dictionaries without type enforcement.

    Tests flexibility in type handling.
    """
    return [{'a': 1, 'b': 2}, {'a': 2, 'b': 3}, {'a': 3, 'b': 4}]


@component
def list_dict_maker_3() -> List:
    """Returns a basic list (no typing or structure guarantees).

    Tests the limits of compiler type handling.
    """
    return [{'a': 1, 'b': 2}, {'a': 2, 'b': 3}, {'a': 3, 'b': 4}]




@dsl.pipeline(name='pipeline-with-loops')
def my_pipeline(loop_parameter: List[str]):
    dict_loop_argument = [{'a': 1, 'b': 2}, {'a': 2, 'b': 3}, {'a': 3, 'b': 4}]


    # with dsl.ParallelFor(items=dict_loop_argument, parallelism=2) as item:
    #     print_text(msg="4")

    with dsl.ParallelFor(items=['1'], parallelism=1) as item:
        print_int(x=item.a)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
