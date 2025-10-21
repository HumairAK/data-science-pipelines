import functools

from kubernetes import client
import kfp
from kfp import dsl
from kfp.dsl import (
    Input,
    Output,
    Artifact,
    Dataset,
    component,
    Metrics,
    Markdown,
    HTML,
    ClassificationMetrics,
)

base_image="quay.io/opendatahub/ds-pipelines-ci-executor-image:v1.0"
dsl.component = functools.partial(dsl.component, base_image=base_image)


@component()
def digit_classification(metrics: Output[Metrics]):

    metrics.log_metric('accuracy', 0.5)


@component()
def wine_classification(metrics: Output[ClassificationMetrics]):
    metrics.log_roc_curve([12, 2.3, 52], [1, 3.3, 5], [3, 1.2, 0.2])


@component()
def iris_sgdclassifier(test_samples_fraction: float, metrics: Output[ClassificationMetrics]):
    pass


@component()
def html_visualization(html_artifact: Output[HTML]):
    html_content = '<!DOCTYPE html><html><body><h1>Hello world</h1></body></html>'
    with open(html_artifact.path, 'w') as f:
        f.write(html_content)


@component()
def markdown_visualization(markdown_artifact: Output[Markdown]):
    markdown_content = '## Hello world \n\n Markdown content'
    with open(markdown_artifact.path, 'w') as f:
        f.write(markdown_content)


@dsl.pipeline(name='metrics-visualization-pipeline')
def metrics_visualization_pipeline():
    wine_classification_op = wine_classification()
    iris_sgdclassifier_op = iris_sgdclassifier(test_samples_fraction=0.3)
    digit_classification_op = digit_classification()
    html_visualization_op = html_visualization()
    markdown_visualization_op = markdown_visualization()


if __name__ == '__main__':
    from kfp import compiler

    compiler.Compiler().compile(
        pipeline_func=metrics_visualization_pipeline,
        package_path=__file__+".yaml"
    )