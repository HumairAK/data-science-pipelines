To install `kfp`, run:

```sh
pip install kfp
```

## Getting started

The following is an example of a simple pipeline that uses the `kfp` v2 syntax:

```python
from kfp import dsl
import kfp


@dsl.component
def add(a: float, b: float) -> float:
    '''Calculates sum of two arguments'''
    return a + b


@dsl.pipeline(
    name='Addition pipeline',
    description='An example pipeline that performs addition calculations.')
def add_pipeline(
    a: float = 1.0,
    b: float = 7.0,
):
    first_add_task = add(a=a, b=4.0)
    second_add_task = add(a=first_add_task.output, b=b)


client = kfp.Client(host='<my-host-url>')
client.create_run_from_pipeline_func(
    add_pipeline, arguments={
        'a': 7.0,
        'b': 8.0
    })

```
