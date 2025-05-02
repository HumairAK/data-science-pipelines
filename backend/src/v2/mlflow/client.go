package mlflow

import (
	"context"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_v2"
)

type MetadataMLFlow struct {
	trackingServerHost string
	experimentID       string
}

func (m *MetadataMLFlow) CreatePipelineRun(ctx context.Context, runName, pipelineName, runID, namespace, runResource, pipelineRoot, storeSessionInfo string) (*metadata_v2.PipelineRun, error) {
	err := m.CreateRun(runName)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (m *MetadataMLFlow) CreateExecution(ctx context.Context, pipeline *metadata_v2.PipelineRun, config *metadata.ExecutionConfig) (*metadata_v2.Execution, error) {
	return nil, nil
}

func NewMetadataMLFlow(h string, e string) (*MetadataMLFlow, error) {
	return &MetadataMLFlow{
		trackingServerHost: h,
		experimentID:       e,
	}, nil
}
