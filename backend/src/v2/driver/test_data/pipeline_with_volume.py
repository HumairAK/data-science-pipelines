import functools

from kfp import dsl
from kfp import kubernetes

base_image="quay.io/opendatahub/ds-pipelines-ci-executor-image:v1.0"
dsl.component = functools.partial(dsl.component, base_image=base_image)


@dsl.component
def producer() -> str:
    with open('/data/file.txt', 'w') as file:
        file.write('Hello world')
    with open('/data/file.txt', 'r') as file:
        content = file.read()
    print(content)
    return content


@dsl.component
def consumer() -> str:
    with open('/data/file.txt', 'r') as file:
        content = file.read()
    print(content)
    assert content == 'Hello world'
    return content


@dsl.pipeline
def pipeline_with_volume():
    pvc1 = kubernetes.CreatePVC(
        pvc_name_suffix='-my-pvc',
        access_modes=['ReadWriteOnce'],
        size='5Mi',
        storage_class_name='standard',
    )

    task1 = producer()
    task2 = consumer().after(task1)

    kubernetes.mount_pvc(
        task1,
        pvc_name=pvc1.outputs['name'],
        mount_path='/data',
    )
    kubernetes.mount_pvc(
        task2,
        pvc_name=pvc1.outputs['name'],
        mount_path='/data',
    )

    delete_pvc1 = kubernetes.DeletePVC(
        pvc_name=pvc1.outputs['name']).after(task2)

if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_volume,
        package_path=__file__.replace('.py', '.yaml'))