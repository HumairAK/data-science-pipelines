from typing import Union

from kfp.kubernetes import kubernetes_executor_config_pb2 as pb
from kfp.dsl import pipeline_channel
from kfp.compiler.pipeline_spec_builder import to_protobuf_value


def parse_k8s_parameter_input(input_param: Union[pipeline_channel.PipelineParameterChannel, str]) -> pb.InputParameterSpec:
    param_spec = pb.InputParameterSpec()

    if isinstance(input_param, str):
        param_spec.runtime_value.constant.CopyFrom(to_protobuf_value(input_param))
    elif isinstance(input_param, pipeline_channel.PipelineParameterChannel):
        if input_param.task_name is None:
            param_spec.component_input_parameter = input_param.full_name
        else:
            param_spec.task_output_parameter.producer_task = input_param.task_name
            param_spec.task_output_parameter.output_parameter_key = input_param.name
    else:
        if input_param.task_name:
            raise ValueError(
                f'Argument for {"input_param"!r} must be an instance of str or PipelineChannel.'
                f'Got unknown input type: {type(input_param)!r}. '
            )

    return param_spec
