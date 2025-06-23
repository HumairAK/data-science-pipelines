package mlflow

import (
	"fmt"
	"github.com/golang/glog"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/mlflow/types"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	corev1 "k8s.io/api/core/v1"
	"strconv"
	"time"
)

// Ensure RunProvider implements RunProvider
var _ metadata_provider.RunProvider = &RunProvider{}

type RunProvider struct {
	client *Client
}

func (r *RunProvider) GetRun(experimentID string, ProviderRunID string) (*metadata_provider.ProviderRun, error) {
	run, err := r.client.getRun(ProviderRunID)
	if err != nil {
		return nil, err
	}
	providerRun := &metadata_provider.ProviderRun{
		ID:     run.Info.RunID,
		Name:   run.Info.RunName,
		Status: string(run.Info.Status),
	}
	return providerRun, nil
}

func (r *RunProvider) CreateRun(
	experimentID string,
	kfpRun *apiv2beta1.Run,
	ProviderRunName string,
	parameters []metadata_provider.RunParameter,
	parentRunID string,
) (*metadata_provider.ProviderRun, error) {
	tags := []types.RunTag{
		{
			Key:   "kfpPipelineRunID",
			Value: kfpRun.RunId,
		},
	}
	if parentRunID != "" {
		tags = append(tags, types.RunTag{
			Key:   "mlflow.parentRunId",
			Value: parentRunID,
		})
	}

	run, err := r.client.createRun(ProviderRunName, tags, experimentID)
	if err != nil {
		return nil, err
	}

	var params []types.Param
	for _, p := range parameters {
		param := types.Param{
			Key:   p.Name,
			Value: p.Value,
		}
		params = append(params, param)
	}
	if len(params) > 0 {
		err = r.client.logBatch(run.Info.RunID, []types.Metric{}, params, []types.RunTag{})
		if err != nil {
			return nil, err
		}
	}

	providerRun := &metadata_provider.ProviderRun{
		ID:     run.Info.RunID,
		Name:   run.Info.RunName,
		Status: string(run.Info.Status),
	}
	return providerRun, nil
}

var mapKFPRuntimeStateToMLFlowRuntimeState = map[model.RuntimeState]types.RunStatus{
	model.RuntimeStateUnspecified: types.Running,
	model.RuntimeStatePending:     types.Scheduled,
	model.RuntimeStateRunning:     types.Running,
	model.RuntimeStateSucceeded:   types.Finished,
	model.RuntimeStateSkipped:     types.Failed,
	model.RuntimeStateFailed:      types.Failed,
	model.RuntimeStateCancelling:  types.Running,
	model.RuntimeStateCanceled:    types.Killed,
	model.RuntimeStatePaused:      types.Running,
}

func ConvertKFPToMLFlowRuntimeState(kfpRunStatus model.RuntimeState) types.RunStatus {
	if v, ok := mapKFPRuntimeStateToMLFlowRuntimeState[kfpRunStatus]; ok {
		return v
	}
	glog.Errorf("Unknown kfp run status: %v", kfpRunStatus)
	return types.Running
}

func (r *RunProvider) UpdateRunStatus(providerRunID string, kfpRunStatus model.RuntimeState) error {
	glog.Infof("Calling UpdateRunStatus with runID %d", providerRunID)

	run, err := r.client.getRun(providerRunID)
	if err != nil {
		return err
	}
	mlflowRunState := ConvertKFPToMLFlowRuntimeState(kfpRunStatus)
	endTime := time.Now().UnixMilli()
	err = r.client.updateRun(run.Info.RunID, nil, &mlflowRunState, &endTime)
	if err != nil {
		return err
	}

	return nil
}

func (r *RunProvider) ExecutorPatch(experimentID string, providerRunID string, storeSession objectstore.SessionInfo) (*corev1.PodSpec, error) {
	var params *objectstore.S3Params
	if storeSession.Provider == "minio" || storeSession.Provider == "s3" {
		var err error
		params, err = objectstore.StructuredS3Params(storeSession.Params)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("Unsupported object store provider: %s", storeSession.Provider)
	}

	mlflowEnvVars := []corev1.EnvVar{
		{
			Name:  "MLFLOW_RUN_ID",
			Value: providerRunID,
		},
		{
			Name:  "MLFLOW_TRACKING_URI",
			Value: r.client.baseHost,
		},
		{
			Name:  "MLFLOW_EXPERIMENT_ID",
			Value: experimentID,
		},

		// ObjectStore Config
		{

			Name:  "MLFLOW_S3_ENDPOINT_URL",
			Value: params.Endpoint,
		},
		{
			Name:  "MLFLOW_S3_IGNORE_TLS",
			Value: strconv.FormatBool(params.DisableSSL),
		},

		{
			Name: "AWS_ACCESS_KEY_ID",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: params.SecretName,
					},
					Key: params.AccessKeyKey,
				},
			},
		},
		{
			Name: "AWS_SECRET_ACCESS_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: params.SecretName,
					},
					Key: params.SecretKeyKey,
				},
			},
		},
	}

	podSpec := &corev1.PodSpec{
		Containers: []corev1.Container{{
			Name: "main",
			Env:  mlflowEnvVars,
		}},
	}

	return podSpec, nil
}
