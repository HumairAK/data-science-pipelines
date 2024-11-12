import os
from typing import NamedTuple, List

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
    return "bob"

@dsl.component
def bye(some_in_b: str):
    return some_in_b


@dsl.component
def crust(ds1: dsl.Input[dsl.Dataset], ds2: dsl.Input[dsl.Dataset]):
    with open(ds1.path, 'r') as f:
        print('ds1: ', f.read())
    with open(ds2.path, 'r') as f:
        print('ds2: ', f.read())

@dsl.component
def max_accuracy(models: dsl.Input[List[dsl.Model]]) -> float:
    return 1.2

@dsl.component
def train(epochs: int, model: dsl.Output[dsl.Model]):
    print('boo')

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

    with dsl.If(h.outputs['someout'] == 'heads'):
        print("somefoo")
        with dsl.If(h.outputs['someout'] == 'heads2'):
            print("somefoo")
            h1= hello(some_in="foo2")

    with dsl.If(h.outputs['someout'] == 'heads3'):
        print("somefoo")
        h2 = hello(some_in="foo3")
        bye(some_in_b=h2.output)

    h3 = hello(some_in="foo3")
    with dsl.ExitHandler(exit_task=h3):
        dag_task2 = inner_pipeline_B()

    with dsl.ParallelFor(items=[1, 5]) as epochs:
        train_model_task = train(epochs=epochs)
    # Will create output + outputdef in create_pipeline_spec for the parallel loop dag
    max_accuracy(models=dsl.Collected(train_model_task.outputs['model']) )

    return dag_task.outputs

# Same as before, and now outer_pipeline component_spec is the final root
@dsl.pipeline()
def outer_pipeline(runtime_param: str) ->  dsl.Dataset:
    dag_task = inner_pipeline_A()

    task = crust(ds1=dag_task.outputs['ds1'], ds2=dag_task.outputs['ds2'])
    return dag_task.outputs['ds1']

if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=outer_pipeline,
        package_path=__file__.replace('.py', '-v2.yaml'))