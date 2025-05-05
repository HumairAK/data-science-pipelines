package metadata_v2

import (
	"context"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
)

type MetadataInterfaceClient interface {
	CreatePipelineRun(ctx context.Context, runName, pipelineName, namespace, runResource, pipelineRoot, storeSessionInfo string, runID int64) (*metadata.Pipeline, error)
	CreateExecution(ctx context.Context, pipeline *metadata.Pipeline, config *metadata.ExecutionConfig, experimentID string) (*metadata.Execution, error)
}
