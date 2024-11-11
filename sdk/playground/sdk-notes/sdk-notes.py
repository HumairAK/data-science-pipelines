from kfp import dsl
from kfp import kubernetes

@dsl.component(base_image="quay.io/opendatahub/ds-pipelines-sample-base:v1.0")
def comp(some_input_arg: str, some_hardcoded_arg: str):
    import os
    print("SECRET_VAR=" + os.environ['SECRET_VAR'])
    print(some_input_arg)
    print(some_hardcoded_arg)

@dsl.pipeline
def pipeline_secret_env(pipeline_input_param: str = "some_default"):

    # Everything in this function is running within a pipeline context
    # as part of the GraphComponent() init
    # First thing it will do is convert pipeline_input_param (and other inputs)
    # Into prameter channels. (Try putting a debugger below, and inspect pipeline_input_param,
    # it will be a parameter channel of type Pipeline Channel)

    # Now, comp is a function decorated with @component, under the hood this is a PythonComponent(BaseComponent)
    # And BaseComponent implements __call__
    # So def comp is actually a PythonComponent CLASS OBJECT (not a function).
    # When you invoke comp(), you are invoking __call__ from the PythonComponent(BaseComponent) which
    # Returns an INITIALIZED Pipeline Task object. The __init__ that's ran in __init__ will (amongst other things)
    # REGISTER the tasks to the PIPELINE CONTEXT's tasks and task groups (see Pipeline Class init & enter calls)
    task = comp(some_input_arg=pipeline_input_param, some_hardcoded_arg="some_hardcoded_value")
    task.set_caching_options(enable_caching=False)

    SECRET_NAME = "user-gcp-sa"
    kubernetes.use_secret_as_env(
        task,
        secret_name=SECRET_NAME,
        secret_key_to_env={'somekey': 'SECRET_VAR'})


# Once we get to this point, all @component and @pipeline functions are no longer functions
# But instead they are instantiated PythonComponent and GraphComponent objects
# So basically we have generated the pipeline spec files (see builder.create_pipeline_spec in graphcomponent __init__)
# And the compiler just converts (over writing things like pipeline name/description/etc as needed)
    # Technically no, the ComponentSpec is created, it's @property pipeline_spec and platform_spec will generate
    # the pb within compile step when componentspec.pipeline_spec etc. is retrieved  (see call of write_pipeline_spec_to_file)
# this to a json/yaml representation.
if __name__ == '__main__':
    # The pipeline_secret_env is wrapped in GraphComponent
    from kfp import compiler
    compiler.Compiler().compile(pipeline_secret_env, 'before.py.yaml')
