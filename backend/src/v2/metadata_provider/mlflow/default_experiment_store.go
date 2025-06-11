package mlflow

import (
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
)

// Ensure MLFlowExperimentProvider implements MetadataExperimentProvider
var _ storage.DefaultExperimentStoreInterface = &DefaultExperimentStore{}

type DefaultExperimentStore struct {
	client *Client
}

func NewDefaultExperimentStore(config *MLFlowServerConfig) (*DefaultExperimentStore, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, err
	}
	return &DefaultExperimentStore{client: client}, nil
}

// Ensure DefaultExperimentStore implements DefaultExperimentStoreInterface
var _ storage.DefaultExperimentStoreInterface = &DefaultExperimentStore{}

func (s *DefaultExperimentStore) GetDefaultExperimentId() (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (s *DefaultExperimentStore) SetDefaultExperimentId(id string) error {
	return fmt.Errorf("not implemented")
}
