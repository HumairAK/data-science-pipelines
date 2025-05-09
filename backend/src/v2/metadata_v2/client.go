package metadata_v2

import (
	"context"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
)

type MetadataInterfaceClient interface {
	CreatePipelineRun(ctx context.Context, runName, pipelineName, namespace, runResource, pipelineRoot, storeSessionInfo string, parentRunID, runID *int64) (*metadata.Pipeline, error)
	CreateExecution(ctx context.Context, pipeline *metadata.Pipeline, config *metadata.ExecutionConfig, experimentID string) (*metadata.Execution, error)
	UpdatePipelineStatus(ctx context.Context, runID *int64, status pb.Execution_State) error
}
