package aimstack

import (
	"context"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_v2"
)

type MetadataAimstack struct {
	repo       string
	experiment string
}

func (m *MetadataAimstack) CreatePipelineRun(ctx context.Context, pipelineName, runID, namespace, runResource, pipelineRoot, storeSessionInfo string) (*metadata_v2.PipelineRun, error) {
	return nil, nil
}

func (m *MetadataAimstack) CreateExecution(ctx context.Context, pipeline *metadata_v2.PipelineRun, config *metadata.ExecutionConfig) (*metadata_v2.Execution, error) {
	return nil, nil
}

func NewMetadataAimstack(r string, e string) (*MetadataAimstack, error) {
	return &MetadataAimstack{
		repo:       r,
		experiment: e,
	}, nil
}
