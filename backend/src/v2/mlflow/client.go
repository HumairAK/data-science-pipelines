package mlflow

import (
	"context"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

type MetadataMLFlow struct {
	trackingServerHost string
	experimentID       string
}

func NewMetadataMLFlow(h string, e string) (*MetadataMLFlow, error) {
	return &MetadataMLFlow{
		trackingServerHost: h,
		experimentID:       e,
	}, nil
}

func (m *MetadataMLFlow) GetPipeline(ctx context.Context, pipelineName, runID, namespace, runResource, pipelineRoot, storeSessionInfo string) (*metadata.Pipeline, error) {
	tags := []map[string]string{{
		"key":   "kfpRunID",
		"value": runID,
	}}
	err := m.CreateRun("some_name", tags)
	if err != nil {
		return nil, err
	}
	return nil, nil

	return nil, nil
}

func (m *MetadataMLFlow) GetDAG(ctx context.Context, executionID int64) (*metadata.DAG, error) {
	return nil, nil
}

func (m *MetadataMLFlow) PublishExecution(
	ctx context.Context, execution *metadata.Execution, outputParameters map[string]*structpb.Value,
	outputArtifacts []*metadata.OutputArtifact, state pb.Execution_State) error {
	return nil
}

func (m *MetadataMLFlow) CreateExecution(ctx context.Context, pipeline *metadata.Pipeline, config *metadata.ExecutionConfig) (*metadata.Execution, error) {
	return nil, nil
}

func (m *MetadataMLFlow) PrePublishExecution(ctx context.Context, execution *metadata.Execution, config *metadata.ExecutionConfig) (*metadata.Execution, error) {
	return nil, nil
}

func (m *MetadataMLFlow) GetExecutions(ctx context.Context, ids []int64) ([]*pb.Execution, error) {
	return nil, nil
}

func (m *MetadataMLFlow) GetExecution(ctx context.Context, id int64) (*metadata.Execution, error) {
	return nil, nil
}

func (m *MetadataMLFlow) GetPipelineFromExecution(ctx context.Context, id int64) (*metadata.Pipeline, error) {
	return nil, nil
}

func (m *MetadataMLFlow) GetExecutionsInDAG(ctx context.Context, dag *metadata.DAG, pipeline *metadata.Pipeline, filter bool) (executionsMap map[string]*metadata.Execution, err error) {
	return nil, nil
}

func (m *MetadataMLFlow) UpdateDAGExecutionsState(ctx context.Context, dag *metadata.DAG, pipeline *metadata.Pipeline) (err error) {
	return nil
}

func (m *MetadataMLFlow) PutDAGExecutionState(ctx context.Context, executionID int64, state pb.Execution_State) (err error) {
	return nil
}

func (m *MetadataMLFlow) GetEventsByArtifactIDs(ctx context.Context, artifactIds []int64) ([]*pb.Event, error) {
	return nil, nil
}

func (m *MetadataMLFlow) GetArtifactName(ctx context.Context, artifactId int64) (string, error) {
	return "", nil
}

func (m *MetadataMLFlow) GetArtifacts(ctx context.Context, ids []int64) ([]*pb.Artifact, error) {
	return nil, nil
}

func (m *MetadataMLFlow) GetOutputArtifactsByExecutionId(ctx context.Context, executionId int64) (map[string]*metadata.OutputArtifact, error) {
	return nil, nil

}

func (m *MetadataMLFlow) RecordArtifact(ctx context.Context, outputName, schema string, runtimeArtifact *pipelinespec.RuntimeArtifact, state pb.Artifact_State, bucketConfig *objectstore.Config) (*metadata.OutputArtifact, error) {
	return nil, nil
}

func (m *MetadataMLFlow) GetOrInsertArtifactType(ctx context.Context, schema string) (typeID int64, err error) {
	return 0, err
}

func (m *MetadataMLFlow) FindMatchedArtifact(ctx context.Context, artifactToMatch *pb.Artifact, pipelineContextId int64) (matchedArtifact *pb.Artifact, err error) {
	return nil, nil
}

// Abstract to interface:

func (m *MetadataMLFlow) GetInputArtifactsByExecutionID(ctx context.Context, executionID int64) (inputs map[string]*pipelinespec.ArtifactList, err error) {
	return nil, nil
}
