package mlflow

import (
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/mlflow/types"
)

// Ensure MLFlowExperimentProvider implements MetadataExperimentProvider
var _ storage.ExperimentStoreInterface = &ExperimentStore{}

// ExperimentStore
// Used by api server creates/manages experiments
// Experiment store will need to check to see if the artifact_location (e.g. in mlflow case)
// is using a bucket in s3 (or something else entirely) that is not configured or supported
// by this kfp deployment.
type ExperimentStore struct {
	client *Client
}

func NewExperimentStore(config *MLFlowServerConfig) (*ExperimentStore, error) {
	// get mlflow client (provider)
	// Problem:
	// API Server needs to create client for mlflow
	// Driver/Launcher need to create client for mlflow
	// API Server needs to pass passthrough connection info for a Provider
	// Launcher/Driver needs to use it to create MLFlow or other client
	client, err := NewClient(config)
	if err != nil {
		return nil, err
	}
	return &ExperimentStore{client: client}, nil
}

// CreateExperiment will need to also accept a providerConfig which is used for
// provider-specific creation options.
// For example, MLFlow allows you to set the artifact_location for all artifacts
// uploaded in any run for a given experiment. The user should be able to configure this.
// If the provider config is not provided, we would use the default bucket path
// TODO: Experiment Store should accet pass through map[string]interface{} for providerConfig
func (s *ExperimentStore) CreateExperiment(baseExperiment *model.Experiment, providerConfig *metadata_provider.ProviderRuntimeConfig) (*model.Experiment, error) {
	experimentTags := []types.ExperimentTag{
		{
			Key:   NamespaceTag,
			Value: baseExperiment.Namespace,
		},
		{
			Key:   ExperimentDescriptionTag,
			Value: baseExperiment.Description,
		},
	}

	creationConfig, err := ConvertToExperimentCreationConfig(*providerConfig)
	if err != nil {
		return nil, err
	}

	mlflowExperimentID, err := s.client.createExperiment(baseExperiment.Name, creationConfig.ArtifactLocation, experimentTags)
	if err != nil {
		return nil, err
	}

	experiment, err := s.client.getExperiment(mlflowExperimentID)
	if err != nil {
		return nil, err
	}

	modelExperiment, err := mlflowExperimentToModelExperiment(*experiment)
	if err != nil {
		return nil, err
	}
	return modelExperiment, nil

}

func (s *ExperimentStore) GetExperiment(uuid string) (*model.Experiment, error) {
	experiment, err := s.client.getExperiment(uuid)
	if err != nil {
		return nil, err
	}
	experimentModel, err := mlflowExperimentToModelExperiment(*experiment)
	if err != nil {
		return nil, err
	}
	return experimentModel, nil
}

// GetExperimentByNameNamespace returns the experiment with the given name and namespace.
// If no experiment is found, it returns an error.
func (s *ExperimentStore) GetExperimentByNameNamespace(name string, namespace string) (*model.Experiment, error) {
	filter := fmt.Sprintf("name='%s' AND %s='%s'", name, NamespaceTag, namespace)
	experiments, err := s.client.searchExperiments(1, "", filter, []string{}, "")
	if err != nil {
		return nil, err
	}
	if len(experiments) < 1 {
		return nil, fmt.Errorf("no experiment found with name %s and namespace %s", name, namespace)
	}
	experiment := experiments[1]
	experimentModel, err := mlflowExperimentToModelExperiment(experiment)
	if err != nil {
		return nil, err
	}
	return experimentModel, nil
}
func (s *ExperimentStore) ListExperiments(filterContext *model.FilterContext, opts *list.Options) ([]*model.Experiment, int, string, error) {
	return nil, 0, "", fmt.Errorf("not implemented")
}
func (s *ExperimentStore) ArchiveExperiment(expId string) error {
	return fmt.Errorf("not implemented")
}
func (s *ExperimentStore) UnarchiveExperiment(expId string) error {
	return fmt.Errorf("not implemented")
}
func (s *ExperimentStore) DeleteExperiment(uuid string) error {
	return fmt.Errorf("not implemented")
}
func (s *ExperimentStore) SetLastRunTimestamp(run *model.Run) error {
	return fmt.Errorf("not implemented")
}
