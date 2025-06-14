package mlflow

import (
	"fmt"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
)

// Ensure RunProvider implements RunProvider
var _ metadata_provider.RunProvider = &RunProvider{}

type RunProvider struct {
	client *Client
}

func (r *RunProvider) GetRun(experimentID string, kfpRunID string) (*metadata_provider.ProviderRun, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *RunProvider) CreateRun(
	experimentID string,
	kfpRun *apiv2beta1.Run,
	parameters []metadata_provider.RunParameter,
	parentRunID string,
) (*metadata_provider.ProviderRun, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *RunProvider) UpdateRunStatus(experimentID string, kfpRunID string, kfpRunStatus model.RuntimeState) error {
	return fmt.Errorf("not implemented")
}
