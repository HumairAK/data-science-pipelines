import os
from typing import NamedTuple

from kfp import Client
from kfp import dsl
from kfp import compiler

print("hit this first")

@dsl.component
def core(core_input: str, ds1: dsl.Output[dsl.Dataset], ds2: dsl.Output[dsl.Dataset]):
    with open(ds1.path, 'w') as f:
        f.write('foo')
    with open(ds2.path, 'w') as f:
        f.write('bar')

@dsl.component
def hello(some_in: str, someout: dsl.Output[str]):
    print("hello")

@dsl.component
def crust(ds1: dsl.Input[dsl.Dataset], ds2: dsl.Input[dsl.Dataset]):
    with open(ds1.path, 'r') as f:
        print('ds1: ', f.read())
    with open(ds2.path, 'r') as f:
        print('ds2: ', f.read())

# First time we hit @pipeline
# GraphComponent is __init__
# pipeline context invoked, pipeline_spec/platform_spec created
# resulting pipeline_spec has this pipeline as root, and core as comp
@dsl.pipeline
def inner_pipeline_B() -> NamedTuple('outputs', ds1=dsl.Dataset, ds2=dsl.Dataset):  # type: ignore
    task = core(core_input="bar")
    return task.outputs

# 2nd time we hit @pipeline
# GraphComponent is __init__
# pipeline context invoked, pipeline_spec/platform_spec created
# resulting pipeline_spec has this pipeline as root
# previous pipeline root shifted to component, and hello is also added to comp section
@dsl.pipeline
def inner_pipeline_A() -> NamedTuple('outputs', ds1=dsl.Dataset, ds2=dsl.Dataset):  # type: ignore
    dag_task = inner_pipeline_B()
    h = hello(some_in="foo")

    with dsl.Condition(h.outputs['someout'] == 'heads'):
        print("somefoo")

    return dag_task.outputs

# same as before, and now outer_pipeline component_spec is the final root
@dsl.pipeline()
def outer_pipeline(runtime_param: str) ->  dsl.Dataset:
    dag_task = inner_pipeline_A()

    task = crust(ds1=dag_task.outputs['ds1'], ds2=dag_task.outputs['ds2'])
    return dag_task.outputs['ds1']

if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=outer_pipeline,
        package_path=__file__.replace('.py', '-v2.yaml'))