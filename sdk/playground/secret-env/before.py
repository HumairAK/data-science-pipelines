from kfp import dsl
from kfp import kubernetes

@dsl.component(base_image="quay.io/opendatahub/ds-pipelines-sample-base:v1.0")
def comp(some_input_arg: str, some_hardcoded_arg: str):
    import os
    print("SECRET_VAR=" + os.environ['SECRET_VAR'])
    print(some_input_arg)
    print(some_hardcoded_arg)

@dsl.component(base_image="quay.io/opendatahub/ds-pipelines-sample-base:v1.0")
def regular_input_comp(some_input_arg: str):
    print(some_input_arg)


@dsl.pipeline
def pipeline_secret_env(pipeline_input_param: str = "some_default"):
    # comp() is basically task -> baseComponent __call__ implementation
    # after comp() is wrapped as a baseComponent class, this comp() is basically a baseComponent() call
    comp.pipeline_spec
    regular_input_comp(some_input_arg=pipeline_input_param)

    task = comp(some_input_arg=pipeline_input_param, some_hardcoded_arg="some_hardcoded_value")
    task.set_caching_options(enable_caching=False)

    kubernetes.use_secret_as_env(
        task,
        secret_name=pipeline_input_param,
        secret_key_to_env={'somekey': 'SECRET_VAR'})


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(pipeline_secret_env, 'before.py.yaml')