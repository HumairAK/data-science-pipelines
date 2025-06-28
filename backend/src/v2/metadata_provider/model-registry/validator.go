package mlflow

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/kubeflow/model-registry/pkg/openapi"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/config"
	corev1 "k8s.io/api/core/v1"
)

// Ensure MLFlowValidator implements Validator
var _ metadata_provider.Validator = &Validator{}

type Validator struct {
	client *Client
}

func (v *Validator) ValidateConfig(config config.GenericProviderConfig) error {
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

	client, err := NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create MLFlow client: %w", err)
	}

	err = client.IsHealthy()
	if err != nil {
		return fmt.Errorf("failed to connect to MLFlow server using the provided configs: %w", err)
	}

	v.client = client
	glog.Info("Successfully connected to MLFlow server using the provided configs")
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
	return nil
}

func (v *Validator) ValidateExperiment(experiment *api.CreateExperimentRequest) error {
	if experiment.Experiment.GetDisplayName() == "" {
		return fmt.Errorf("experiment name is empty")
	}
	// TODO: Enable once form is added to the UI
	//if experiment.Experiment.ProviderConfig == nil {
	//	return fmt.Errorf("experiment provider config is empty")
	//}
	return nil
}

func validateModelRegistryExperiment(experiment *openapi.Experiment, namespace string) error {
	// todo
	return nil
}
