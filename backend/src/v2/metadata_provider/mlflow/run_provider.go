package mlflow

import (
	"github.com/golang/glog"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/mlflow/types"
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

	// TODO: switch to batch update
	for _, param := range parameters {
		err = r.client.logParam(run.Info.RunID, param.Name, param.Value)
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
	model.RuntimeStatePending:   types.Scheduled,
	model.RuntimeStateRunning:   types.Running,
	model.RuntimeStateSucceeded: types.Finished,
	model.RuntimeStateSkipped:   types.Failed,
	model.RuntimeStateFailed:    types.Failed,
	model.RuntimeStateCanceled:  types.Killed,
	model.RuntimeStatePaused:    types.Running,
}

func (r *RunProvider) UpdateRunStatus(experimentID string, ProviderRunID string, kfpRunStatus model.RuntimeState) error {
	glog.Infof("Calling UpdateRunStatus with runID %d", ProviderRunID)

	run, err := r.client.getRun(ProviderRunID)
	if err != nil {
		return err
	}

	mlflowRunState := mapKFPRuntimeStateToMLFlowRuntimeState[kfpRunStatus]
	var endTime *int64

	err = r.client.updateRun(run.Info.RunID, nil, &mlflowRunState, endTime)
	if err != nil {
		return err
	}

	return nil
}

func (r *RunProvider) NestedRunsSupported() bool {
	return true
}
