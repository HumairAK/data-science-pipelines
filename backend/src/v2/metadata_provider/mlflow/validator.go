package mlflow

import (
	"fmt"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/mlflow/types"
)

// Ensure MLFlowValidator implements Validator
var _ metadata_provider.Validator = &Validator{}

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

func validateMLFlowExperiment(experiment types.Experiment, namespace string) error {
	// if namespace is set, then we expect the experiment to:
	// have a namespace tag with the same value as the namespace
	if namespace != "" {
		if GetExperimentTag(&experiment, NamespaceTag) != namespace {
			return fmt.Errorf("namespace tag in experiment %s does not match namespace %s", experiment.ExperimentID, namespace)
		}
		if experiment.Name != BuildExperimentNamespaceName(experiment.Name, namespace) {
			return fmt.Errorf("experiment name %s with namespace %s, does not match the format <namespace>/<kfp_experiment_name>", experiment.Name, namespace)
		}
	}

	return nil
}
