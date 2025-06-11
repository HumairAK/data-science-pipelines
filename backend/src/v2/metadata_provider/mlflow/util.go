package mlflow

import (
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/mlflow/types"
)

func mlflowExperimentToModelExperiment(experiment types.Experiment) (*model.Experiment, error) {
	experimentModel := &model.Experiment{
		UUID:        experiment.ExperimentID,
		Name:        experiment.Name,
		Description: GetExperimentTag(&experiment, ExperimentDescriptionTag),
		// Unix time value (used by MLFlow) is in milliseconds,
		// but the google.protobuf.Timestamp struct expects seconds in its Seconds field.
		CreatedAtInSec: experiment.CreationTime / 1000,
		// TODO: This is not an exact mapping and can be misleading
		// last update makes more sense and model.experiment should support that
		// lastrun is used to sort kfp ui experiments by last run
		LastRunCreatedAtInSec: experiment.LastUpdateTime / 1000,
		Namespace:             GetExperimentTag(&experiment, NamespaceTag),
	}

	switch experiment.LifecycleStage {
	case "active":
		experimentModel.StorageState = model.StorageStateAvailable
	case "deleted":
		experimentModel.StorageState = model.StorageStateArchived
	default:
		experimentModel.StorageState = model.StorageStateUnspecified
	}
	return experimentModel, nil
}

// GetExperimentTag return value associated with tag or empty string if tag doesn't exist
func GetExperimentTag(experiment *types.Experiment, tagKey string) string {
	for _, tag := range experiment.Tags {
		if tag.Key == tagKey {
			return tag.Value
		}
	}
	return ""
}

func BuildExperimentNamespaceName(name, namespace string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}
