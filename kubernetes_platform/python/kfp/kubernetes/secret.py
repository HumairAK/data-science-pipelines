# Copyright 2023 The Kubeflow Authors
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

from typing import Dict

from google.protobuf import json_format

from kfp.compiler.pipeline_spec_builder import to_protobuf_value
from kfp.dsl import PipelineTask, pipeline_channel
from kfp.kubernetes import common
from kfp.kubernetes import kubernetes_executor_config_pb2 as pb


def use_secret_as_env(
    task: PipelineTask,
    secret_name: str,
    secret_key_to_env: Dict[str, str],
) -> PipelineTask:
    """Use a Kubernetes Secret as an environment variable as described by the `Kubernetes documentation
    https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets-as-environment-variables `_.

    Args:
        task: Pipeline task.
        secret_name: Name of the Secret.
        secret_key_to_env: Dictionary of Secret data key to environment variable name. For example, ``{'password': 'PASSWORD'}`` sets the data of the Secret's password field to the environment variable ``PASSWORD``.

    Returns:
        Task object with updated secret configuration.
    """

    msg = common.get_existing_kubernetes_config_as_message(task)

    key_to_env = [
        pb.SecretAsEnv.SecretKeyToEnvMap(
            secret_key=secret_key,
            env_var=env_var,
        ) for secret_key, env_var in secret_key_to_env.items()
    ]
    # Find out how we go from pipeline inputs to task inputs for channels
    # there will be some area where @pipeline inputs, are fed into task spec inputs somehow
    # this is probably in the to .pb conversion logic I think, replicate that here

    secret_name_parameter = pb.InputParameterSpec()
    secret_as_env = pb.SecretAsEnv(key_to_env=key_to_env)
    if isinstance(secret_name, pipeline_channel.PipelineParameterChannel):
        secret_name_parameter.component_input_parameter = secret_name.full_name
    elif isinstance(secret_name, str):
        secret_as_env.secret_name = secret_name
        secret_name_parameter.runtime_value.constant.CopyFrom(to_protobuf_value(secret_name))
    else:
        raise ValueError(
            'Secret name supports only the following types: '
            'str or parameter channel'
            f'Got {secret_name} of type {type(secret_name)}.')
    secret_as_env.secret_name_parameter.CopyFrom(secret_name_parameter)

    msg.secret_as_env.append(secret_as_env)

    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)

    return task


def use_secret_as_volume(
    task: PipelineTask,
    secret_name: str,
    mount_path: str,
    optional: bool = False,
) -> PipelineTask:
    """Use a Kubernetes Secret by mounting its data to the task's container as
    described by the `Kubernetes documentation <https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets-as-files-from-a-pod>`_.

    Args:
        task: Pipeline task.
        secret_name: Name of the Secret.
        mount_path: Path to which to mount the Secret data.
        optional: Optional field specifying whether the Secret must be defined.

    Returns:
        Task object with updated secret configuration.
    """

    msg = common.get_existing_kubernetes_config_as_message(task)

    secret_as_vol = pb.SecretAsVolume(
        secret_name=secret_name,
        mount_path=mount_path,
        optional=optional,
    )

    msg.secret_as_volume.append(secret_as_vol)

    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)

    return task
