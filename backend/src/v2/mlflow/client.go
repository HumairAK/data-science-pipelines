package mlflow

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_v2"
	"github.com/kubeflow/pipelines/backend/src/v2/mlflow/types"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"os"
	"strconv"
	"time"
)

const (
	MlflowParentRunId = "mlflow.parentRunId" // Parent DAG Execution ID.
)

// metadata keys
const (
	keyDisplayName           = "display_name"
	keyTaskName              = "task_name"
	keyImage                 = "image"
	keyPodName               = "pod_name"
	keyPodUID                = "pod_uid"
	keyNamespace             = "namespace"
	keyResourceName          = "resource_name"
	keyPipelineRoot          = "pipeline_root"
	keyStoreSessionInfo      = "store_session_info"
	keyCacheFingerPrint      = "cache_fingerprint"
	keyCachedExecutionID     = "cached_execution_id"
	keyInputs                = "inputs"
	keyOutputs               = "outputs"
	keyParameterProducerTask = "parameter_producer_task"
	keyOutputArtifacts       = "output_artifacts"
	keyArtifactProducerTask  = "artifact_producer_task"
	keyParentDagID           = "parent_dag_id" // Parent DAG Execution ID.
	keyIterationIndex        = "iteration_index"
	keyIterationCount        = "iteration_count"
	keyTotalDagTasks         = "total_dag_tasks"
)

type MetadataMLFlow struct {
	trackingServerHost string
}

// CreatePipelineRun
// runID is the Run Id of the pipeline runc reated by api server
// we use this to retrieve the parent run, this won't work for nested pipeline case probably
// ParentRunID is optional, an empty parentRunID means this is not a nested Pipeline.
func (m *MetadataMLFlow) CreatePipelineRun(ctx context.Context, runName, pipelineName, namespace, runResource, pipelineRoot, storeSessionInfo, experimentID string, parentRunID, runID *int64) (*metadata.Pipeline, error) {

	tags := []types.RunTag{
		{
			Key:   "kfpRunID",
			Value: strconv.FormatInt(*runID, 10),
		},
		{
			Key:   "keyNamespace",
			Value: namespace,
		},
		{
			Key:   "keyResourceName",
			Value: runResource,
		},
		{
			Key:   "keyPipelineRoot",
			Value: metadata.GenerateOutputURI(pipelineRoot, []string{pipelineName, strconv.FormatInt(*runID, 10)}, true),
		},
		{
			Key:   "keyStoreSessionInfo",
			Value: storeSessionInfo,
		},
	}

	if parentRunID != nil {
		// For now parentID is the mlmd execution id which is stored as a TAG on the corresponding mlflow parent ID
		// so we need to use this tag to fetch the MLFLOW parent RUN ID. The assumption right now is that
		// there is only ever one mlflow run with a tag `kfpRunID: 61`, but for testing there may be multiple
		// so we'll sort by cretion time descending and pick the first one (i.e. latest).
		filterQuery := fmt.Sprintf("tags.kfpRunID = '%d'", *parentRunID)
		runs, err := m.searchRuns(
			[]string{experimentID},
			1,
			"",
			filterQuery,
			[]string{"start_time DESC"},
			types.ACTIVE_ONLY,
		)
		if err != nil {
			return nil, err
		}
		if len(runs) <= 0 {
			return nil, fmt.Errorf("no runs found with parentRunID %d", *parentRunID)
		}
		parentRun := runs[0]

		tags = append(tags, types.RunTag{
			Key:   "mlflow.parentRunId",
			Value: parentRun.Info.RunID,
		})
	}
	_, err := m.createRun(runName, tags, experimentID)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (m *MetadataMLFlow) UpdatePipelineStatus(ctx context.Context, runID *int64, status pb.Execution_State, experimentID string) error {
	glog.Infof("Calling UpdatePipelineStatus with runID %d, with status: %d", runID, status)

	filterQuery := fmt.Sprintf("tags.kfpRunID = '%d'", *runID)
	runs, err := m.searchRuns(
		[]string{experimentID},
		1,
		"",
		filterQuery,
		[]string{"start_time DESC"},
		types.ACTIVE_ONLY,
	)
	if err != nil {
		return err
	}
	if len(runs) <= 0 {
		return fmt.Errorf("No run found with given id %d", *runID)
	}

	var mlflowRunState types.RunStatus
	var endTime *int64
	switch status {
	case pb.Execution_RUNNING:
		mlflowRunState = types.Running
		break
	case pb.Execution_FAILED:
		mlflowRunState = types.Failed
		t := time.Now().UnixMilli()
		endTime = &t
	case pb.Execution_COMPLETE:
		mlflowRunState = types.Finished
		t := time.Now().UnixMilli()
		endTime = &t
	}

	run := runs[0]
	_, err = m.updateRun(run.Info.RunID, nil, &mlflowRunState, endTime)
	if err != nil {
		return err
	}

	return nil
}

func (m *MetadataMLFlow) CreateExperiment(ctx context.Context, experimentName, experimentDescription, kfpExperimentID string) (*string, error) {
	experimentTags := []types.ExperimentTag{
		{
			Key:   "kfpExperimentID",
			Value: kfpExperimentID,
		},
		{
			Key:   "mlflow.note.content",
			Value: experimentDescription,
		},
	}
	mlflowExperimentID, err := m.createExperiment(experimentName, experimentTags)
	if err != nil {
		return nil, err
	}

	return mlflowExperimentID, nil
}

func (m *MetadataMLFlow) GetExperiment(ctx context.Context, kfpExperimentID string) (*metadata_v2.Experiment, error) {
	experiment := &metadata_v2.Experiment{}

	glog.Infof("Calling GetExperiment with kfpExperimentID %s", kfpExperimentID)

	filterQuery := fmt.Sprintf("tags.kfpExperimentID = '%s'", kfpExperimentID)
	experiments, err := m.searchExperiments(
		1,
		"",
		filterQuery,
		[]string{"start_time DESC"},
		types.ACTIVE_ONLY,
	)
	if err != nil {
		return nil, err
	}
	if len(experiments) <= 0 {
		return nil, fmt.Errorf("No experiments found with given id %s", kfpExperimentID)
	}
	mlFlowExperiment := experiments[0]
	experiment.Name = mlFlowExperiment.Name
	experiment.Description = ""
	for _, tag := range mlFlowExperiment.Tags {
		if tag.Key == "mlflow.note.content" {
			experiment.Description = tag.Value
		}
	}
	return experiment, nil
}

func NewMetadataMLFlow() (*MetadataMLFlow, error) {
	hostEnv := os.Getenv("MLFLOW_HOST")
	portEnv := os.Getenv("MLFLOW_PORT")
	tlsEnabled := os.Getenv("MLFLOW_TLS_ENABLED")

	var protocol string
	if tlsEnabled == "true" {
		protocol = "https"
	} else {
		protocol = "http"
	}
	var host string
	if portEnv != "" {
		host = fmt.Sprintf("%s://%s:%s/api/2.0/mlflow", protocol, hostEnv, portEnv)
	} else {
		host = fmt.Sprintf("%s://%s/api/2.0/mlflow", protocol, hostEnv)
	}

	return &MetadataMLFlow{
		trackingServerHost: host,
	}, nil
}
