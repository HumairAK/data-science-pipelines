package manager

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

type Provider struct {
	config ProviderConfig
}

func NewProvider(config ProviderConfig) (*Provider, error) {
	provider := &Provider{config: config}
	err := provider.ValidateConfig()
	if err != nil {
		return nil, err
	}
	return provider, nil
}

func NewProviderFromJSON(raw string) (*Provider, error) {
	var cfg ProviderConfig
	err := json.Unmarshal([]byte(raw), &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata provider config, error: %w\nmetadataProviderConfig: %v", err, raw)
	}
	return NewProvider(cfg)
}

type ProviderConfig struct {
	MetadataProviderName MetadataProvider        `json:"MetadataProviderName"`
	EnvironmentVariables []k8score.EnvVar        `json:"EnvironmentVariables"`
	Config               common.UnstructuredJSON `json:"Config"`
}

func (p *Provider) NewExperimentStore() (storage.ExperimentStoreInterface, error) {
	factory, ok := metadata_provider.Lookup(string(p.config.MetadataProviderName))
	if !ok {
		return nil, fmt.Errorf("unsupported metadata provider: %s", p.config.MetadataProviderName)
	}
	return factory.NewExperimentStore(p.config.Config)
}

func (p *Provider) NewRunProvider() (metadata_provider.RunProvider, error) {
	factory, ok := metadata_provider.Lookup(string(p.config.MetadataProviderName))
	if !ok {
		return nil, fmt.Errorf("unsupported metadata provider: %s", p.config.MetadataProviderName)
	}
	return factory.NewRunProvider(p.config.Config)
}

func (p *Provider) NewMetadataArtifactProvider() (metadata_provider.MetadataArtifactProvider, error) {
	factory, ok := metadata_provider.Lookup(string(p.config.MetadataProviderName))
	if !ok {
		return nil, fmt.Errorf("unsupported metadata provider: %s", p.config.MetadataProviderName)
	}
	return factory.NewMetadataArtifactProvider(p.config.Config)
}

func (p *Provider) NewValidator() (metadata_provider.Validator, error) {
	factory, ok := metadata_provider.Lookup(string(p.config.MetadataProviderName))
	if !ok {
		return nil, fmt.Errorf("unsupported metadata provider: %s", p.config.MetadataProviderName)
	}
	return factory.NewValidator(p.config.Config)
}

func (p *Provider) ValidateConfig() error {
	if p == nil {
		return fmt.Errorf("metadata provider config is empty")
	}
	if p.config.MetadataProviderName == "" {
		return fmt.Errorf("metadata provider name is empty")
	}
	if p.config.Config != nil {
		factory, ok := metadata_provider.Lookup(string(p.config.MetadataProviderName))
		if !ok {
			return fmt.Errorf("unsupported metadata provider: %s", p.config.MetadataProviderName)
		}
		validator, err := factory.NewValidator(p.config.Config)
		if err != nil {
			return err
		}
		err = validator.ValidateConfig(p.config.Config, p.config.EnvironmentVariables)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Provider) ProviderConfigToJSON() (string, error) {
	jsonBytes, err := json.Marshal(p.config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata provider config, error: %w\nmetadataProviderConfig: %v", err, p)
	}
	return string(jsonBytes), nil
}

func (p *Provider) GetEnvVars() []k8score.EnvVar {
	return p.config.EnvironmentVariables
}
