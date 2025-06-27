package mlflow

import (
	"fmt"
	"github.com/kubeflow/model-registry/pkg/openapi"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/mlflow/types"
	"strconv"
)

func mrExperimentToModelExperiment(experiment openapi.Experiment) (*model.Experiment, error) {
	nsValue := experiment.GetCustomProperties()["namespace"]
	ns := nsValue.MetadataStringValue.GetStringValue()
	createdAtInSec, err := strconv.ParseInt(experiment.GetCreateTimeSinceEpoch(), 10, 64)
	if err != nil {
		return nil, err
	}
	lastRunCreatedAtInSec, err := strconv.ParseInt(experiment.GetLastUpdateTimeSinceEpoch(), 10, 64)
	if err != nil {
		return nil, err
	}
	experimentModel := &model.Experiment{
		UUID:        *experiment.Id,
		Name:        experiment.Name,
		Description: *experiment.Description,
		// Unix time value (used by MLFlow) is in milliseconds,
		// but the google.protobuf.Timestamp struct expects seconds in its Seconds field.
		CreatedAtInSec: createdAtInSec / 1000,
		// TODO: This is not an exact mapping and can be misleading
		// last update makes more sense and model.experiment should support that
		// lastrun is used to sort kfp ui experiments by last run
		LastRunCreatedAtInSec: lastRunCreatedAtInSec / 1000,
		Namespace:             ns,
	}

	switch experiment.GetState() {
	case openapi.EXPERIMENTSTATE_LIVE:
		experimentModel.StorageState = model.StorageStateAvailable
	case openapi.EXPERIMENTSTATE_ARCHIVED:
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
