package mlflow

import (
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
)

type mlflowFactory struct{}

func (f *mlflowFactory) NewValidator(cfg common.UnstructuredJSON) (metadata_provider.Validator, error) {
	return NewMLFlowValidator(cfg)
}

func (f *mlflowFactory) NewExperimentStore(cfg common.UnstructuredJSON) (storage.ExperimentStoreInterface, error) {
	return NewExperimentStore(cfg)
}

func (f *mlflowFactory) NewRunProvider(cfg common.UnstructuredJSON) (metadata_provider.RunProvider, error) {
	return NewRunsProvider(cfg)
}

func (f *mlflowFactory) NewMetadataArtifactProvider(cfg common.UnstructuredJSON) (metadata_provider.MetadataArtifactProvider, error) {
	return NewArtifactsProvider(cfg)
}

func init() {
	metadata_provider.Register("mlflow", &mlflowFactory{})
}
