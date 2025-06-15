package mlflow

import (
	"fmt"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
	"google.golang.org/protobuf/types/known/structpb"
)

// Ensure ArtifactProvider implements MetadataArtifactsProvider
var _ metadata_provider.MetadataArtifactProvider = &ArtifactProvider{}

type ArtifactProvider struct {
	client *Client
}

func (a *ArtifactProvider) LogOutputArtifact(
	runID string,
	runtimeArtifact *pipelinespec.RuntimeArtifact,
) (*metadata_provider.ArtifactResult, error) {
	var artifactResult *metadata_provider.ArtifactResult

	switch runtimeArtifact.Name {
	case "metrics":
		for key, value := range runtimeArtifact.Metadata.GetFields() {
			var valueType *structpb.Value

			switch value.Kind.(type) {
			case *structpb.Value_NumberValue:
				valueType = value
			}

			if valueType == nil {
				return nil, fmt.Errorf("no number value found for metric type")
			}
			err := a.client.logMetric(runID, runID, key, valueType.GetNumberValue())
			if err != nil {
				return nil, err
			}
		}
	default:
		return artifactResult, nil
	}

	if artifactResult == nil {
		return nil, fmt.Errorf("artifact result is nil")
	}
	return artifactResult, nil
}

func (a *ArtifactProvider) NestedRunsSupported() bool {
	return true
}
