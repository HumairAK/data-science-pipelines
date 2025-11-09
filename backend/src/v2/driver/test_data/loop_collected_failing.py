import functools
from typing import List

from kfp import dsl
from kfp.dsl import (
    Input,
    Output,
    Artifact,
    Dataset,
    component
)

base_image="quay.io/opendatahub/ds-pipelines-ci-executor-image:v1.0"
dsl.component = functools.partial(dsl.component, base_image=base_image)

@component
def process_dataset(
        model_id_in: str,
        output_artifact: Output[Artifact],
):
    with open(output_artifact.path, "w") as f:
        data_out = f"{model_id_in}"
        f.write(data_out)
        print(data_out)
    output_artifact.metadata["model_id"] = model_id_in

@component
def analyze_artifact(analyze_artifact_input: Input[Artifact], analyze_output_artifact: Output[Artifact]):
    with open(analyze_artifact_input.path, "r") as f:
        data = f.read()
    with open(analyze_output_artifact.path, "w") as f:
        f.write(f'{{"values": {data}}}')
    if data == '3':
        print("Failing 3rd task")
        exit(1)
    print("task succeeded: " + data)

@dsl.pipeline
def primary_pipeline():
    with dsl.ParallelFor(items=['1', '2', '3']) as model_id:
        process_dataset_task = process_dataset(model_id_in=model_id)
        analyze_artifact(analyze_artifact_input=process_dataset_task.outputs["output_artifact"])


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=primary_pipeline,
        package_path=__file__ + ".yaml"
    )
