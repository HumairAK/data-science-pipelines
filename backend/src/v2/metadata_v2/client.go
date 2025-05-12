package metadata_v2

import (
	"context"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
)

type Experiment struct {
	ID          string
	Name        string
	Description string
}

type MetadataInterfaceClient interface {
	CreatePipelineRun(ctx context.Context, runName, pipelineName, namespace, runResource, pipelineRoot, storeSessionInfo, experimentID string, mlmdParentRunID, mlmdExecutionID *int64) (*metadata.Pipeline, error)
	UpdatePipelineStatus(ctx context.Context, mlmdExecutionID *int64, status pb.Execution_State, experimentID string) error
	CreateExperiment(ctx context.Context, experimentName, experimentDescription, kfpExperimentID string) (*string, error)
	GetExperiment(ctx context.Context, kfpExperimentID string) (*Experiment, error)
	// LogRunMetric returns Metric URI
	LogRunMetric(ctx context.Context, experimentID string, mlmdExecutionID int64, metricName string, metricValue float64) (*string, error)
	LogParameter(ctx context.Context, experimentID string, mlmdExecutionID int64, parameterName string, parameterValue string) error
	//LogInput(ctx context.Context, experimentID string, mlmdExecutionID int64, name, digest, sourceType, source, schema, profile string) error
	//LogModel(ctx context.Context, experimentID string, mlmdExecutionID int64, parameterName string, parameterValue string) error
}
