package experiments

import (
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
)

// Ensure MLFlowExperimentProvider implements MetadataExperimentProvider
var _ storage.ExperimentStoreInterface = &MLFlowExperimentStore{}

// MLFlowExperimentStore
// Used by api server creates/manages experiments
// Experiment store will need to check to see if the artifact_location (e.g. in mlflow case)
// is using a bucket in s3 (or something else entirely) that is not configured or supported
// by this kfp deployment.
type MLFlowExperimentStore struct {
	apiPath     string
	baseHost    string
	metricsPath string
}

// CreateExperiment will need to also accept a providerConfig which is used for
// provider-specific creation options.
// For example, MLFlow allows you to set the artifact_location for all artifacts
// uploaded in any run for a given experiment. The user should be able to configure this.
// If the provider config is not provided, we would use the default bucket path
// TODO: Experiment Store should accet pass through map[string]interface{} for providerConfig
func (s *MLFlowExperimentStore) CreateExperiment(experiment *model.Experiment) (*model.Experiment, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *MLFlowExperimentStore) GetExperiment(uuid string) (*model.Experiment, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *MLFlowExperimentStore) GetExperimentByNameNamespace(name string, namespace string) (*model.Experiment, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *MLFlowExperimentStore) ListExperiments(filterContext *model.FilterContext, opts *list.Options) ([]*model.Experiment, int, string, error) {
	return nil, 0, "", fmt.Errorf("not implemented")
}
func (s *MLFlowExperimentStore) ArchiveExperiment(expId string) error {
	return fmt.Errorf("not implemented")
}
func (s *MLFlowExperimentStore) UnarchiveExperiment(expId string) error {
	return fmt.Errorf("not implemented")
}
func (s *MLFlowExperimentStore) DeleteExperiment(uuid string) error {
	return fmt.Errorf("not implemented")
}
func (s *MLFlowExperimentStore) SetLastRunTimestamp(run *model.Run) error {
	return fmt.Errorf("not implemented")
}
