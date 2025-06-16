package mlflow

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
	"google.golang.org/protobuf/types/known/structpb"
)

// Ensure ArtifactProvider implements MetadataArtifactsProvider
var _ metadata_provider.MetadataArtifactProvider = &ArtifactProvider{}

type SchemaTitle string

const (
	SchemaTitleArtifacts              = "system.Artifact"
	SchemaTitleModel                  = "system.Model"
	SchemaTitleMetrics                = "system.Metrics"
	SchemaTitleDataset                = "system.Dataset"
	SchemaTitleClassificationMetrics  = "system.ClassificationMetrics"
	SchemaSlicedClassificationMetrics = "system.SlicedClassificationMetrics"
	SchemaHTML                        = "system.HTML"
	SchemaMarkdown                    = "system.Markdown"
)

type ArtifactProvider struct {
	client *Client
}

func (a *ArtifactProvider) LogOutputArtifact(
	runID string,
	experimentID string,
	runtimeArtifact *pipelinespec.RuntimeArtifact,
) (*metadata_provider.ArtifactResult, error) {
	var artifactResult *metadata_provider.ArtifactResult
	schemaTitle := runtimeArtifact.Type.GetSchemaTitle()
	switch schemaTitle {
	case SchemaTitleMetrics:
		var foundValue *structpb.Value
		var foundKey *string
		for key, value := range runtimeArtifact.Metadata.GetFields() {
			switch value.Kind.(type) {
			case *structpb.Value_NumberValue:
				foundValue = value
				key = key
			}
		}
		if foundKey == nil || foundValue == nil {
			return nil, fmt.Errorf("no number key/value found for metric type")
		}
		err := a.client.logMetric(runID, runID, *foundKey, foundValue.GetNumberValue())
		if err != nil {
			return nil, err
		}
		artifactResult.Name = runtimeArtifact.Name
		artifactResult.ArtifactURI = metricURI(runID, experimentID, *foundKey, a.client.GetMetricsPath())
		artifactResult.ArtifactURL = artifactResult.ArtifactURI
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

func metricURI(runId, experimentID, metricKey, metricsPath string) string {
	uri := fmt.Sprintf("%s?runs=%%5B%%22%s%%22%%5D&experiments=%%5B%%22%s%%22%%5D&metric=%%22%s%%22&plot_metric_keys=%%5B%%22%s%%22%%5D",
		metricsPath, runId, experimentID, metricKey, metricKey)
	glog.Infof("built uri: %s", uri)
	return uri
}
