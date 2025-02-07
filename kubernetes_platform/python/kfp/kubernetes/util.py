from typing import Union

from kfp.kubernetes import kubernetes_executor_config_pb2 as pb
from kfp.dsl import pipeline_channel
from kfp.compiler.pipeline_spec_builder import to_protobuf_value


def parse_k8s_parameter_input(input_param: Union[pipeline_channel.PipelineParameterChannel, str]) -> pb.InputParameterSpec:
    param_spec = pb.InputParameterSpec()
    if isinstance(input_param, pipeline_channel.PipelineParameterChannel):
        if input_param.task_name:
            raise ValueError("encountered upstream task parameter input"
                             "kubernetes_platform input parameters currently "
                             "only support pipeline inputs and not upstream"
                             "task inputs.")
        param_spec.component_input_parameter = input_param.full_name
    elif isinstance(input_param, str):
        param_spec.runtime_value.constant.CopyFrom(to_protobuf_value(input_param))
    else:
        raise ValueError(
            'input supports only the following types: '
            'str or parameter channel'
            f'Got {input_param} of type {type(input_param)}.')
    return param_spec
