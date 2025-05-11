package mlflow

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
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
	experimentID       string
}

// CreatePipelineRun
// runID is the Run Id of the pipeline runc reated by api server
// we use this to retrieve the parent run, this won't work for nested pipeline case probably
// ParentRunID is optional, an empty parentRunID means this is not a nested Pipeline.
func (m *MetadataMLFlow) CreatePipelineRun(ctx context.Context, runName, pipelineName, namespace, runResource, pipelineRoot, storeSessionInfo string, parentRunID, runID *int64) (*metadata.Pipeline, error) {

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
		runs, err := m.SearchRuns(
			[]string{m.experimentID},
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
	_, err := m.CreateRun(runName, tags)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (m *MetadataMLFlow) CreateExecution(ctx context.Context, pipeline *metadata.Pipeline, config *metadata.ExecutionConfig, experimentID string) (*metadata.Execution, error) {
	glog.Infof("Calling CreateExecution with pipeline %d, with experimentID: %s", pipeline.GetRunCtxID(), experimentID)

	// Create nested execution run
	if config.ParentDagID != 0 {
		return nil, fmt.Errorf("ParentDag Id required")
	}

	KFPParentRunID := strconv.FormatInt(config.ParentDagID, 10)
	filterString := fmt.Sprintf("tags.kfpRunID = '%s'", KFPParentRunID)
	// Get MLFLow run id by searching parentID
	runs, err := m.SearchRuns([]string{experimentID}, 10, "", filterString, []string{"DESC"}, "")
	if err != nil {
		return nil, err
	}

	// should be the first (and only) run:
	if len(runs) == 0 {
		return nil, fmt.Errorf("No runs found for experiment %s", experimentID)
	} else if len(runs) > 1 {
		return nil, fmt.Errorf("Multiple runs found for experiment %s", experimentID)
	}

	ParentMLFlowRun := runs[0]

	// This will connect the two runs in a parent -> child relationship
	// in MLFlow
	tags := []types.RunTag{
		{
			Key:   MlflowParentRunId,
			Value: ParentMLFlowRun.Info.RunID,
		},
		{
			Key:   keyPodName,
			Value: config.PodName,
		},
		{
			Key:   keyNamespace,
			Value: config.Namespace,
		},
		{
			Key:   keyPipelineRoot,
			Value: pipeline.GetPipelineRoot(),
		},
		{
			Key:   keyDisplayName,
			Value: config.TaskName,
		},
	}

	createdRun, err := m.CreateRun(config.Name, tags)
	if err != nil {
		return nil, err
	}

	// Log params once run is created
	// would be nice if we could log these at creation time
	// to avoid multiple calls but I didn't see a way to
	// do this.
	if config.InputParameters != nil {
		for key, val := range config.InputParameters {
			info := createdRun.Info
			err := m.LogParam(info.RunID, info.RunUUID, key, fmt.Sprintf("%v", val.AsInterface()))
			if err != nil {
				return nil, err
			}
		}
	}

	glog.Infof("CreateExecution successfully completed, with pipeline %d, with experimentID: %s", pipeline.GetRunCtxID(), experimentID)

	return nil, nil
}

func (m *MetadataMLFlow) UpdatePipelineStatus(ctx context.Context, runID *int64, status pb.Execution_State) error {
	glog.Infof("Calling UpdatePipelineStatus with runID %d, with status: %d", runID, status)

	filterQuery := fmt.Sprintf("tags.kfpRunID = '%d'", *runID)
	runs, err := m.SearchRuns(
		[]string{m.experimentID},
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
	_, err = m.UpdateRun(run.Info.RunID, nil, &mlflowRunState, endTime)
	if err != nil {
		return err
	}

	return nil
}

func NewMetadataMLFlow() (*MetadataMLFlow, error) {
	hostEnv := os.Getenv("MLFLOW_HOST")
	portEnv := os.Getenv("MLFLOW_PORT")
	tlsEnabled := os.Getenv("MLFLOW_TLS_ENABLED")

	experimentID := os.Getenv("MLFLOW_EXPERIMENT_ID")
	if experimentID == "" {
		return nil, fmt.Errorf("Missing experiment ID")
	}

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
		experimentID:       experimentID,
	}, nil
}
