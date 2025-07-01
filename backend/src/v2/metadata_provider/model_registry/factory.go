package model_registry

import (
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/config"
)

func init() {
	metadata_provider.Register("model-registry", &Factory{})
}

type Factory struct{}

func (f *Factory) NewValidator(cfg config.GenericProviderConfig) (metadata_provider.Validator, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &Validator{client: client}, nil
}

func (f *Factory) NewExperimentStore(cfg config.GenericProviderConfig) (storage.ExperimentStoreInterface, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &ExperimentStore{client: client}, nil
}

func (f *Factory) NewRunProvider(cfg config.GenericProviderConfig) (metadata_provider.RunProvider, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &RunProvider{client: client}, nil
}

func (f *Factory) NewMetadataArtifactProvider(cfg config.GenericProviderConfig) (metadata_provider.MetadataArtifactProvider, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &ArtifactProvider{client: client}, nil
}
