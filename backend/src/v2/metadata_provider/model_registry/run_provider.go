package model_registry

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/kubeflow/model-registry/pkg/openapi"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/mlflow"
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
		ID:     *run.Id,
		Name:   *run.Name,
		Status: string(*run.Status),
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
	tags := make(map[string]openapi.MetadataValue)
	for _, param := range parameters {
		mv := openapi.NewMetadataStringValueWithDefaults()
		mv.SetStringValue(param.Value)
		tags[param.Name] = openapi.MetadataValue{
			MetadataStringValue: mv,
		}
	}
	if parentRunID != "" {
		addTag(&tags, mlflow.ParentTag, parentRunID)
	}
	addTag(&tags, RunID, kfpRun.RunId)
	run, err := r.client.createRun(
		ProviderRunName,
		&tags,
		kfpRun.Description,
		experimentID,
		parentRunID,
	)
	if err != nil {
		return nil, err
	}
	providerRun := &metadata_provider.ProviderRun{
		ID:     *run.Id,
		Name:   *run.Name,
		Status: string(*run.Status),
	}
	return providerRun, nil
}

// TODO can model registry support more states?
var mapKFPRuntimeStateToMRRuntimeState = map[model.RuntimeState]openapi.ExperimentRunStatus{
	model.RuntimeStateUnspecified: openapi.EXPERIMENTRUNSTATUS_SCHEDULED,
	model.RuntimeStatePending:     openapi.EXPERIMENTRUNSTATUS_SCHEDULED,
	model.RuntimeStateRunning:     openapi.EXPERIMENTRUNSTATUS_RUNNING,
	model.RuntimeStateSucceeded:   openapi.EXPERIMENTRUNSTATUS_FINISHED,
	model.RuntimeStateSkipped:     openapi.EXPERIMENTRUNSTATUS_FINISHED,
	model.RuntimeStateFailed:      openapi.EXPERIMENTRUNSTATUS_FAILED,
	model.RuntimeStateCancelling:  openapi.EXPERIMENTRUNSTATUS_RUNNING,
	model.RuntimeStateCanceled:    openapi.EXPERIMENTRUNSTATUS_KILLED,
	model.RuntimeStatePaused:      openapi.EXPERIMENTRUNSTATUS_SCHEDULED,
}

func ConvertKFPToMRRuntimeState(kfpRunStatus model.RuntimeState) openapi.ExperimentRunStatus {
	if v, ok := mapKFPRuntimeStateToMRRuntimeState[kfpRunStatus]; ok {
		return v
	}
	glog.Errorf("Unknown kfp run status: %v", kfpRunStatus)
	return openapi.EXPERIMENTRUNSTATUS_RUNNING
}

func (r *RunProvider) UpdateRunStatus(providerRunID string, kfpRunStatus model.RuntimeState) error {
	glog.Infof("Calling UpdateRunStatus with runID %d", providerRunID)

	run, err := r.client.getRun(providerRunID)
	if err != nil {
		return err
	}
	mrRunState := ConvertKFPToMRRuntimeState(kfpRunStatus)
	endTime := time.Now().UnixMilli()
	err = r.client.updateRun(*run.Id, &mrRunState, &endTime, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (r *RunProvider) ExecutorPatch(
	experimentID string,
	providerRunID string,
	storeSession objectstore.SessionInfo,
	pipelineRoot string, env []corev1.EnvVar) (*corev1.PodSpec, error) {
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

	envVars := []corev1.EnvVar{

		// Support MLFlow Env vars for end users
		// This allows easy integration options with MR plugin with MLFlow
		{
			Name:  "MLFLOW_RUN_ID",
			Value: providerRunID,
		},

		{
			Name:  "MLFLOW_TRACKING_URI",
			Value: r.client.Host,
		},
		{
			Name:  "MLFLOW_EXPERIMENT_ID",
			Value: experimentID,
		},
		{
			Name:  "MLFLOW_S3_ENDPOINT_URL",
			Value: params.Endpoint,
		},
		{
			Name:  "MLFLOW_S3_IGNORE_TLS",
			Value: strconv.FormatBool(params.DisableSSL),
		},

		// Support Model Registry Env vars for end users
		{
			Name:  "MODEL_REGISTRY_TRACKING_URI",
			Value: r.client.Host,
		},
		{
			Name:  "MODEL_REGISTRY_ARTIFACT_URI",
			Value: pipelineRoot,
		},
		{
			Name:  "MODEL_REGISTRY_RUN_ID",
			Value: providerRunID,
		},
		{
			Name:  "MODEL_REGISTRY_EXPERIMENT_ID",
			Value: experimentID,
		},

		// ObjectStore URI
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
			Env:  envVars,
		}},
	}

	return podSpec, nil
}
