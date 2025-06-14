package factory

import (
	"encoding/json"
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
	_ "github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/import_providers"
	k8score "k8s.io/api/core/v1"
)

// MetadataProvider defines the type of metadata provider
type MetadataProvider string

type ProviderConfig struct {
	MetadataProviderName MetadataProvider        `json:"MetadataProviderName"`
	EnvironmentVariables []k8score.EnvVar        `json:"EnvironmentVariables"`
	Config               common.UnstructuredJSON `json:"Config"`
}

func (c *ProviderConfig) NewExperimentStore() (storage.ExperimentStoreInterface, error) {
	factory, ok := metadata_provider.Lookup(string(c.MetadataProviderName))
	if !ok {
		return nil, fmt.Errorf("unsupported metadata provider: %s", c.MetadataProviderName)
	}
	return factory.NewExperimentStore(c.Config)
}

func (c *ProviderConfig) NewRunProvider() (metadata_provider.RunProvider, error) {
	factory, ok := metadata_provider.Lookup(string(c.MetadataProviderName))
	if !ok {
		return nil, fmt.Errorf("unsupported metadata provider: %s", c.MetadataProviderName)
	}
	return factory.NewRunProvider(c.Config)
}

func (c *ProviderConfig) NewMetadataArtifactProvider() (metadata_provider.MetadataArtifactProvider, error) {
	factory, ok := metadata_provider.Lookup(string(c.MetadataProviderName))
	if !ok {
		return nil, fmt.Errorf("unsupported metadata provider: %s", c.MetadataProviderName)
	}
	return factory.NewMetadataArtifactProvider(c.Config)
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

func (c *ProviderConfig) ProviderConfigToJSON() (string, error) {
	jsonBytes, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata provider config, error: %w\nmetadataProviderConfig: %v", err, c)
	}
	return string(jsonBytes), nil
}

func JSONToProviderConfig(jsonSTR string) (*ProviderConfig, error) {
	var metadataProviderConfig ProviderConfig
	err := json.Unmarshal([]byte(jsonSTR), &metadataProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata provider config, error: %w\nmetadataProviderConfig: %v", err, jsonSTR)
	}
	return &metadataProviderConfig, nil
}
