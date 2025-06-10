package util

import (
	"context"
	v2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
)

func CreateRunMetadata(
	ctx context.Context,
	providerRunName string,
	cm *client_manager.ClientManager,
	experimentID string,
	runID string,
	ecfg *metadata.ExecutionConfig,
	parentID string,
) (ProviderRunId string, err error) {
	rsc := cm.RunServiceClient()
	kfpRun, err := rsc.GetRun(
		ctx,
		&v2beta1.GetRunRequest{
			RunId: runID,
		})
	if err != nil {
		return "", err
	}
	run, err := cm.MetadataRunProvider().CreateRun(
		experimentID,
		kfpRun,
		providerRunName,
		metadata_provider.PBParamsToRunParameters(ecfg.InputParameters),
		parentID,
	)
	if err != nil {
		return "", err
	}
	ecfg.ProviderRunID = &run.ID
	return run.ID, nil
}

var mlmdExecutionStateToKFPState = map[pb.Execution_State]model.RuntimeState{
	pb.Execution_UNKNOWN:  model.RuntimeStateUnspecified,
	pb.Execution_NEW:      model.RuntimeStatePending,
	pb.Execution_RUNNING:  model.RuntimeStateRunning,
	pb.Execution_COMPLETE: model.RuntimeStateSucceeded,
	pb.Execution_FAILED:   model.RuntimeStateFailed,
	pb.Execution_CACHED:   model.RuntimeStateSucceeded,
	pb.Execution_CANCELED: model.RuntimeStateCanceled,
}

func GetKFPStateFromMLMDState(state pb.Execution_State) model.RuntimeState {
	if kfpState, ok := mlmdExecutionStateToKFPState[state]; ok {
		return kfpState
	}
	return model.RuntimeStateUnspecified
}
