package mlflow

import (
	"encoding/json"
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/config"
)

type Config struct {
	Host       string `json:"Host"`
	Port       string `json:"Port"`
	TLSEnabled string `json:"TLSEnabled"`
	Debug      bool   `json:"Debug"`
	// TODO: Add tls cert handling
}

type ExperimentCreationConfig struct {
	// not implemented
}

func ConvertToExperimentCreationConfig(config config.GenericProviderConfig) (*ExperimentCreationConfig, error) {
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

func ConvertToModelRegistryConfig(data config.GenericProviderConfig) (*Config, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal unstructured JSON: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal into MLFlowConfig: %w", err)
	}

	return &cfg, nil
}
