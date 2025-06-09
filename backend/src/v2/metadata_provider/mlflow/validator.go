package mlflow

import (
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
)

// Ensure MLFlowValidator implements MetadataProviderValidator
var _ metadata_provider.MetadataProviderValidator = &Validator{}

type Validator struct {
	client Client
}

func NewMLFlowValidator(client Client) *Validator {
	return &Validator{client: client}
}

func (v *Validator) ValidateRun(kfpRun *api.CreateRunRequest) error {
	// MLFlow doesn't have specific run validation requirements
	return nil
}

func (v *Validator) ValidateExperiment(experiment *api.CreateExperimentRequest) error {
	// MLFlow doesn't have specific experiment validation requirements
	return nil
}
