from kfp import Client
from kfp import compiler


def submit_pipeline(pipeline_path: str, run_name: str, run_desc: str = ""):
    """
    Submit a pipeline to Kubeflow Pipelines platform.
    
    Args:
        pipeline_path: Path to the pipeline definition file
        run_name: Name of the pipeline
        run_desc: Description of the pipeline
    """
    client = Client(
        host='http://localhost:8888',
        verify_ssl=False,
    )

    # Create or get pipeline
    pipeline = client.create_run_from_pipeline_package(
        run_name=run_name,
        pipeline_file=pipeline_path,
    )

    return pipeline


if __name__ == "__main__":
    # Example usage
    PIPELINE_PATH = "metrics.py.yaml"
    RUN_NAME = "Run-Test"
    RUN_DESC = "Some Description"

    submit_pipeline(pipeline_path=PIPELINE_PATH, run_name=RUN_NAME, run_desc=RUN_DESC)
