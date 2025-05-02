package metadata_v2

import (
	"context"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
)

type PipelineRun struct {
	Id           *int64
	Name         *string
	PipelineName *string
}

type Execution struct {
	Id              *int64
	PipelineRunName *string
	PipelineName    *string
}

type MetadataInterfaceClient interface {
	CreatePipelineRun(ctx context.Context, runName, pipelineName, runID, namespace, runResource, pipelineRoot, storeSessionInfo string) (*PipelineRun, error)
	CreateExecution(ctx context.Context, pipeline *PipelineRun, config *metadata.ExecutionConfig) (*Execution, error)
}
