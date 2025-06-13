package mlflow

import (
	"fmt"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/util"
)

// Ensure ArtifactProvider implements MetadataArtifactsProvider
var _ metadata_provider.MetadataArtifactProvider = &ArtifactProvider{}

type ArtifactProvider struct {
	client *Client
}

func NewArtifactsProvider(config util.UnstructuredJSON) (metadata_provider.MetadataArtifactProvider, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, err
	}
	return &ArtifactProvider{client: client}, nil
}

func (a *ArtifactProvider) LogOutputArtifact(
	experimentID string,
	runID string,
	runtimeArtifact *pipelinespec.RuntimeArtifact,
	defaultArtifactURI string,
) (metadata_provider.ArtifactResult, error) {
	return metadata_provider.ArtifactResult{}, fmt.Errorf("not implemented")
}
