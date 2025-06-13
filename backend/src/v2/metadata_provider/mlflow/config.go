package mlflow

import (
	"encoding/json"
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
)

type Config struct {
	Host       string `json:"Host"`
	Port       string `json:"Port"`
	TLSEnabled string `json:"TLSEnabled"`
	// TODO: Add tls cert handling
}

type ExperimentCreationConfig struct {
	ArtifactLocation string `json:"artifact_location"`
}

func ConvertToExperimentCreationConfig(config common.UnstructuredJSON) (*ExperimentCreationConfig, error) {
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

func ConvertToMLFlowConfig(data common.UnstructuredJSON) (*Config, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal unstructured JSON: %w", err)
	}
	var config Config
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal into MLFlowConfig: %w", err)
	}

	return &config, nil
}
