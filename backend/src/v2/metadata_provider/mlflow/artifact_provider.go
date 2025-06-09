package mlflow

import (
	"fmt"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
)

// Ensure ArtifactProvider implements MetadataArtifactsProvider
var _ metadata_provider.MetadataArtifactProvider = &ArtifactProvider{}

type ArtifactProvider struct {
	client Client
}

func NewArtifactsProvider(client Client) *ArtifactProvider {
	return &ArtifactProvider{client: client}
}

func (a *ArtifactProvider) LogOutputArtifact(
	experimentID string,
	runID string,
	runtimeArtifact *pipelinespec.RuntimeArtifact,
	defaultArtifactURI string,
) (metadata_provider.ArtifactResult, error) {
	return metadata_provider.ArtifactResult{}, fmt.Errorf("not implemented")
}
