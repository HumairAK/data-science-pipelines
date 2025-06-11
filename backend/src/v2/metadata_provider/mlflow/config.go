package mlflow

import (
	"encoding/json"
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
	k8score "k8s.io/api/core/v1"
)

type MLFlowConfig struct {
	MLFlowServerConfig *MLFlowServerConfig
}
type ExecutionConfig struct {
	ExperimentID string           `json:"experiment_id"`
	EnvVars      []k8score.EnvVar `json:"env_vars"`
}

type MLFlowServerConfig struct {
	Host       string `json:"host"`
	Port       string `json:"port"`
	TLSEnabled string `json:"TLSEnabled"`
	// TODO: Add tls cert handling
}

type ExperimentCreationConfig struct {
	ArtifactLocation string `json:"artifact_location"`
}

func ConvertToExperimentCreationConfig(config metadata_provider.ProviderRuntimeConfig) (*ExperimentCreationConfig, error) {
	// Re-marshal the generic map to JSON
	bytes, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ProviderRuntimeConfig: %w", err)
	}

	// Unmarshal into the strongly typed struct
	var typed ExperimentCreationConfig
	err = json.Unmarshal(bytes, &typed)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal into ExperimentCreationConfig: %w", err)
	}

	return &typed, nil
}
