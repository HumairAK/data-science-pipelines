import os
from typing import NamedTuple

from kfp import Client
from kfp import dsl
from kfp import compiler


@dsl.component
def core(ds1: dsl.Output[dsl.Dataset], ds2: dsl.Output[dsl.Dataset]):
    with open(ds1.path, 'w') as f:
        f.write('foo')
    with open(ds2.path, 'w') as f:
        f.write('bar')

@dsl.component
def hello():
    print("hello")


@dsl.component
def crust(ds1: dsl.Input[dsl.Dataset], ds2: dsl.Input[dsl.Dataset]):
    with open(ds1.path, 'r') as f:
        print('ds1: ', f.read())
    with open(ds2.path, 'r') as f:
        print('ds2: ', f.read())


@dsl.pipeline
def inner_pipeline_B() -> NamedTuple('outputs', ds1=dsl.Dataset, ds2=dsl.Dataset):  # type: ignore
    task = core()
    return task.outputs


@dsl.pipeline
def inner_pipeline_A() -> NamedTuple('outputs', ds1=dsl.Dataset, ds2=dsl.Dataset):  # type: ignore
    dag_task = inner_pipeline_B()
    h = hello()
    return dag_task.outputs


@dsl.pipeline()
def outer_pipeline(runtime_param: str):
    dag_task = inner_pipeline_A()
    task = crust(ds1=dag_task.outputs['ds1'], ds2=dag_task.outputs['ds2'])


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=outer_pipeline,
        package_path=__file__.replace('.py', '-v2.yaml'))