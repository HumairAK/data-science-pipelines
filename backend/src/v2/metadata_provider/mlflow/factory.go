package mlflow

import (
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/config"
)

type mlflowFactory struct{}

func (f *mlflowFactory) NewValidator(cfg config.GenericProviderConfig) (metadata_provider.Validator, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &Validator{client: client}, nil
}

func (f *mlflowFactory) NewExperimentStore(cfg config.GenericProviderConfig) (storage.ExperimentStoreInterface, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &ExperimentStore{client: client}, nil
}

func (f *mlflowFactory) NewRunProvider(cfg config.GenericProviderConfig) (metadata_provider.RunProvider, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &RunProvider{client: client}, nil
}

func (f *mlflowFactory) NewMetadataArtifactProvider(cfg config.GenericProviderConfig) (metadata_provider.MetadataArtifactProvider, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &ArtifactProvider{client: client}, nil
}

func init() {
	metadata_provider.Register("mlflow", &mlflowFactory{})
}
