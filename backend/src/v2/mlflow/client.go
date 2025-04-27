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
	apiPath     string
	baseHost    string
	metricsPath string
}

// CreatePipelineRun
// runID is the Run Id of the pipeline runc reated by api server
// we use this to retrieve the parent run, this won't work for nested pipeline case probably
// ParentRunID is optional, an empty parentRunID means this is not a nested Pipeline.
func (m *MetadataMLFlow) CreatePipelineRun(ctx context.Context, runName, pipelineName, namespace, runResource, pipelineRoot, storeSessionInfo, experimentID string, mlmdParentRunID, mlmdExecutionID *int64) (*metadata.Pipeline, error) {

	tags := []types.RunTag{
		{
			Key:   "kfpRunID",
			Value: strconv.FormatInt(*mlmdExecutionID, 10),
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
			Value: metadata.GenerateOutputURI(pipelineRoot, []string{pipelineName, strconv.FormatInt(*mlmdExecutionID, 10)}, true),
		},
		{
			Key:   "keyStoreSessionInfo",
			Value: storeSessionInfo,
		},
	}

	if mlmdParentRunID != nil {
		// For now parentID is the mlmd execution id which is stored as a TAG on the corresponding mlflow parent ID
		// so we need to use this tag to fetch the MLFLOW parent RUN ID. The assumption right now is that
		// there is only ever one mlflow run with a tag `kfpRunID: 61`, but for testing there may be multiple
		// so we'll sort by cretion time descending and pick the first one (i.e. latest).
		filterQuery := fmt.Sprintf("tags.kfpRunID = '%d'", *mlmdParentRunID)
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
			return nil, fmt.Errorf("no runs found with mlmdParentRunID %d", *mlmdParentRunID)
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
		[]string{"creation_time DESC"},
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
	experiment.ID = mlFlowExperiment.ExperimentID
	experiment.Description = ""
	for _, tag := range mlFlowExperiment.Tags {
		if tag.Key == "mlflow.note.content" {
			experiment.Description = tag.Value
		}
	}
	return experiment, nil
}

func (m *MetadataMLFlow) LogRunMetric(ctx context.Context, experimentID string, mlmdExecutionID int64, metricName string, metricValue float64) (*string, error) {
	glog.Infof("LogRunMetric called with experimentID %s, mlmdExecutionID: %d, metricName: %s, metricValue: %f", experimentID, mlmdExecutionID, metricName, metricValue)

	glog.Infof("Getting mlflow Run from kfpRunID")
	mlFlowRunId, err := m.getMLFlowRunFromKFPRunID(experimentID, strconv.FormatInt(mlmdExecutionID, 10))
	if err != nil {
		return nil, err
	}
	glog.Infof("Got kfpRunID: %s", *mlFlowRunId)
	glog.Infof("logging metric....")
	err = m.logMetric(*mlFlowRunId, experimentID, metricName, metricValue)
	if err != nil {
		return nil, err
	}
	glog.Infof("metric logged")
	glog.Infof("fetching URI...")

	uri, err := m.metricURI(*mlFlowRunId, experimentID, metricName)
	if err != nil {
		return nil, err
	}
	glog.Infof("URI fetched: %s", *uri)

	// Also log the metric to parent
	glog.Infof("Logging to parent mlflow runs...")
	parentID, err := m.getParentID(*mlFlowRunId)
	if err != nil {
		return nil, err
	}
	glog.Infof("Logging to parent mlflow run: %s", *parentID)
	err = m.logMetric(*parentID, experimentID, metricName, metricValue)
	if err != nil {
		return nil, err
	}

	return uri, nil
}

func (m *MetadataMLFlow) getParentID(mlFlowRunId string) (*string, error) {
	run, err := m.getRun(mlFlowRunId)
	if err != nil {
		return nil, err
	}
	var parentID string
	for _, tag := range run.Data.Tags {
		if tag.Key == "mlflow.parentRunId" {
			parentID = tag.Value
		}
	}
	return &parentID, nil
}

func (m *MetadataMLFlow) LogParameter(ctx context.Context, experimentID string, mlmdExecutionID int64, parameterName string, parameterValue string) error {
	glog.Infof("LogParameter called with experimentID: %s, mlmdExecutionID: %d, parameterName: %s, parameterValue: %s",
		experimentID, mlmdExecutionID, parameterName, parameterValue)
	runID, err := m.getMLFlowRunFromKFPRunID(experimentID, strconv.FormatInt(mlmdExecutionID, 10))
	if err != nil {
		return err
	}
	err = m.logParam(runID, parameterName, parameterValue)
	if err != nil {
		return err
	}
	glog.Infof("Parameter %s logged", parameterName)
	return nil
}

func GetExperimentIDFromEnv() (*string, error) {
	experimentID := os.Getenv("MLFLOW_EXPERIMENT_ID")
	if experimentID == "" {
		return nil, fmt.Errorf("environment variable MLFLOW_EXPERIMENT_ID is not set")
	}

	return &experimentID, nil
}

func (m *MetadataMLFlow) metricURI(runId, experimentID, metricKey string) (*string, error) {
	uri := fmt.Sprintf("%s?runs=%%5B%%22%s%%22%%5D&experiments=%%5B%%22%s%%22%%5D&metric=%%22%s%%22&plot_metric_keys=%%5B%%22%s%%22%%5D",
		m.metricsPath, runId, experimentID, metricKey, metricKey)
	glog.Infof("built uri: %s", uri)
	return &uri, nil
}

func (m *MetadataMLFlow) getMLFlowRunFromKFPRunID(experimentID, kfpExecutionID string) (*string, error) {
	filterQuery := fmt.Sprintf("tags.kfpRunID = '%s'", kfpExecutionID)
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
		return nil, fmt.Errorf("No run found with given id %s", kfpExecutionID)
	}
	return &runs[0].Info.RunID, nil
}

func NewMetadataMLFlow() (*MetadataMLFlow, error) {
	hostEnv := os.Getenv("MLFLOW_HOST")
	portEnv := os.Getenv("MLFLOW_PORT")
	tlsEnabled := os.Getenv("MLFLOW_TLS_ENABLED")
	if hostEnv == "" {
		return nil, fmt.Errorf("Missing environment variable MLFLOW_HOST")
	}

	var protocol string
	if tlsEnabled == "true" {
		protocol = "https"
	} else {
		protocol = "http"
	}
	var basePath string
	if portEnv != "" {
		basePath = fmt.Sprintf("%s://%s:%s", protocol, hostEnv, portEnv)
	} else {
		basePath = fmt.Sprintf("%s://%s", protocol, hostEnv)
	}
	apiPath := fmt.Sprintf("%s/api/2.0/mlflow", basePath)
	metricsPath := fmt.Sprintf("%s/#/metric", basePath)

	return &MetadataMLFlow{
		apiPath:     apiPath,
		baseHost:    basePath,
		metricsPath: metricsPath,
	}, nil
}
