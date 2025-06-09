package mlflow

import (
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
)

// Ensure RunsProvider implements MetadataRunProvider
var _ metadata_provider.MetadataRunProvider = &RunsProvider{}

type RunsProvider struct {
	client MlflowClient
}

func NewRunsProvider(client MlflowClient) *RunsProvider {
	return &RunsProvider{client: client}
}

func (r *RunsProvider) GetRun(experimentID string, kfpRunID string) (*metadata_provider.ProviderRun, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *RunsProvider) CreateRun(
	experimentID string,
	kfpRun model.Run,
	parameters []metadata_provider.RunParameter,
) (*metadata_provider.ProviderRun, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *RunsProvider) LinkParentChildRuns(parentProviderRunID string, childProviderRunID string) error {
	return fmt.Errorf("not implemented")
}

func (r *RunsProvider) UpdateRunStatus(experimentID string, kfpRunID string, kfpRunStatus model.RuntimeState) error {
	return fmt.Errorf("not implemented")
}
