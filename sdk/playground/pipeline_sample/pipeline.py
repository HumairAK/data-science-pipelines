from kfp import dsl

@dsl.component
def inner_comp(dataset: dsl.Output[dsl.Dataset]):
    with open(dataset.path, "w") as f:
        f.write("foobar")


@dsl.component
def outer_comp(input: dsl.Dataset) -> dsl.Model:
    print("input: ", input)
    outer_comp_output = "/some/path"
    return outer_comp_output


@dsl.pipeline
def inner_pipeline(inner_pipeline_param: str) -> dsl.Dataset:
    inner_comp_task = inner_comp()
    return inner_comp_task.output


@dsl.pipeline
def outer_pipeline(run_time_arg: str) -> dsl.Model:
    inner_pipeline_task = inner_pipeline(run_time_arg)
    outer_comp_task = outer_comp(input=inner_pipeline_task.output)
    return outer_comp_task.output


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(outer_pipeline, 'outer_pipeline.yaml')
