import functools
import os
from typing import NamedTuple

from kfp import compiler, dsl
from kfp.dsl import Dataset, Input, Model, component, importer

base_image="quay.io/opendatahub/ds-pipelines-ci-executor-image:v1.0"
dsl.component = functools.partial(dsl.component, base_image=base_image)

@component()
def train(dataset: Input[Dataset]) -> str:
    with open(dataset.path, 'r') as f:
        data = f.read()

    return data


@dsl.pipeline(name='pipeline-with-importer')
def pipeline_with_importer(uri: str = 's3://ml-pipeline/shakespeare1.txt'):
    importer1 = importer(
        artifact_uri='s3://ml-pipeline/shakespeare1.txt',
        artifact_class=Dataset,
        reimport=False)
    train(dataset=importer1.output)

    importer2 = importer(
        artifact_uri=uri,
        artifact_class=Dataset,
        reimport=False)
    train(dataset=importer2.output)


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_importer,
        package_path=__file__.replace('.py', '.yaml'))

