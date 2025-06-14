package mlflow

import (
	"encoding/json"
	"fmt"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/mlflow/types"
	corev1 "k8s.io/api/core/v1"
)

// Ensure MLFlowValidator implements Validator
var _ metadata_provider.Validator = &Validator{}

type Validator struct {
	client *Client
}

func NewMLFlowValidator(config common.UnstructuredJSON) (metadata_provider.Validator, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, err
	}
	return &Validator{client: client}, nil
}

func (v *Validator) ValidateConfig(config common.UnstructuredJSON, envvars []corev1.EnvVar) error {
	var cfg Config
	bytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal unstructured JSON: %w", err)
	}
	err = json.Unmarshal(bytes, &cfg)
	if err != nil {
		return fmt.Errorf("failed to unmarshal into MLFlowConfig: %w", err)
	}
	if cfg.Host == "" {
		return fmt.Errorf("MLFlow host is empty")
	}
	if err := envVarExists("MLFLOW_TRACKING_URI", envvars); err != nil {
		return fmt.Errorf("MLFLOW_TRACKING_URI environment variable not found")
	}

	return nil
}

func envVarExists(key string, envvars []corev1.EnvVar) (err error) {
	for _, env := range envvars {
		if env.Name == key {
			return nil
		}
	}
	return fmt.Errorf("environment variable %s not found", key)
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
		kfpName := GetExperimentTag(&experiment, NameTag)
		if experiment.Name != BuildExperimentNamespaceName(kfpName, namespace) {
			return fmt.Errorf("experiment name %s with namespace %s, does not match the format <namespace>/<kfp_experiment_name>", experiment.Name, namespace)
		}
	}
	return nil
}
