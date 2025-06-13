package config

import (
	"encoding/json"
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/mlflow"
	k8score "k8s.io/api/core/v1"
)

// MetadataProvider defines the type of metadata provider
type MetadataProvider string

const (
	MetadataProviderMLFlow MetadataProvider = "mlflow"
)

type ProviderConfig struct {
	MetadataProviderName MetadataProvider        `json:"MetadataProviderName"`
	EnvironmentVariables []k8score.EnvVar        `json:"EnvironmentVariables"`
	Config               common.UnstructuredJSON `json:"Config"`
}

func (c *ProviderConfig) NewExperimentStore() (storage.ExperimentStoreInterface, error) {
	switch c.MetadataProviderName {
	case MetadataProviderMLFlow:
		store, err := mlflow.NewExperimentStore(c.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to create mlflow experiment store: %w", err)
		}
		return store, nil
	default:
		return nil, fmt.Errorf("unsupported metadata provider: %s", c.MetadataProviderName)
	}
}

func (c *ProviderConfig) NewRunProvider() (metadata_provider.RunProvider, error) {
	switch c.MetadataProviderName {
	case MetadataProviderMLFlow:
		provider, err := mlflow.NewRunsProvider(c.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to create mlflow run provider: %w", err)
		}
		return provider, nil
	default:
		return nil, fmt.Errorf("unsupported metadata provider: %s", c.MetadataProviderName)
	}
}

func (c *ProviderConfig) NewMetadataArtifactProvider() (metadata_provider.MetadataArtifactProvider, error) {
	switch c.MetadataProviderName {
	case MetadataProviderMLFlow:
		provider, err := mlflow.NewArtifactsProvider(c.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to create mlflow artifact provider: %w", err)
		}
		return provider, nil
	default:
		return nil, fmt.Errorf("unsupported metadata provider: %s", c.MetadataProviderName)
	}
}

func (c *ProviderConfig) ValidateConfig() error {
	if c == nil {
		return fmt.Errorf("metadata provider config is empty")
	}
	if c.MetadataProviderName == "" {
		return fmt.Errorf("metadata provider name is empty")
	}
	if c.Config == nil {
		return fmt.Errorf("metadata provider config is empty")
		// todo: add more validation for each provider
	}
	return nil
}

func JSONToProviderConfig(jsonSTR string) (*ProviderConfig, error) {
	var metadataProviderConfig ProviderConfig
	err := json.Unmarshal([]byte(jsonSTR), &metadataProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata provider config, error: %w\nmetadataProviderConfig: %v", err, jsonSTR)
	}
	return &metadataProviderConfig, nil
}
