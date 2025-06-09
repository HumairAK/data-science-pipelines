package mlflow

import (
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
)

// Ensure MLFlowValidator implements MetadataProviderValidator
var _ metadata_provider.MetadataProviderValidator = &MLFlowValidator{}

type MLFlowValidator struct {
	client MlflowClient
}

func NewMLFlowValidator(client MlflowClient) *MLFlowValidator {
	return &MLFlowValidator{client: client}
}

func (v *MLFlowValidator) ValidateRun(kfpRun *api.CreateRunRequest) error {
	// MLFlow doesn't have specific run validation requirements
	return nil
}

func (v *MLFlowValidator) ValidateExperiment(experiment *api.CreateExperimentRequest) error {
	// MLFlow doesn't have specific experiment validation requirements
	return nil
}
