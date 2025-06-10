package import_providers

// Add your provider here to trigger init()
import (
	_ "github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/mlflow"
)