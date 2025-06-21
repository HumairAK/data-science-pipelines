package mlflow

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/mlflow/types"
	"google.golang.org/protobuf/types/known/structpb"
	"net/url"
	"path"
)

// Ensure ArtifactProvider implements MetadataArtifactsProvider
var _ metadata_provider.MetadataArtifactProvider = &ArtifactProvider{}

type SchemaTitle string

type ArtifactProvider struct {
	client *Client
}

func (a *ArtifactProvider) LogOutputArtifact(
	runID string,
	experimentID string,
	runtimeArtifact *pipelinespec.RuntimeArtifact,
) (*metadata_provider.ArtifactResult, error) {
	glog.V(4).Infof("LogOutputArtifact: %v", runtimeArtifact.String())
	var artifactResult metadata_provider.ArtifactResult

	schemaTitle := runtimeArtifact.Type.GetSchemaTitle()
	switch schemaTitle {
	case util.SchemaTitleMetric:
		var metrics []types.Metric
		for key, value := range runtimeArtifact.Metadata.GetFields() {
			switch value.Kind.(type) {
			case *structpb.Value_NumberValue:
				metric := types.Metric{
					Key:   key,
					Value: value.GetNumberValue(),
				}
				metrics = append(metrics, metric)
			}
		}
		if len(metrics) <= 0 {
			return nil, fmt.Errorf("Encountered metrics artifact type, but detected no metadata fields defining numbered values.")
		}
		err := a.client.logBatch(runID, metrics, []types.Param{}, []types.RunTag{})
		if err != nil {
			return nil, err
		}
		// Since there are multiple metrics, there is no way to tie multie URIs
		// to a single KFP artifact, so we just return the Metrics path for MLFlow UI
		url := a.metricURL(runID, experimentID)
		artifactResult.ArtifactURI = url
		artifactResult.ArtifactURL = url
		return &artifactResult, nil
	case util.SchemaTitleArtifact:
		//
		experiment, err := a.client.getExperiment(experimentID)
		if err != nil {
			return nil, err
		}
		// TODO: are we setting the default on api server when it's not set when experiment is creation? Confirm.
		artifactLocation := experiment.ArtifactLocation

		parsed, err := url.Parse(runtimeArtifact.Uri)
		if err != nil {
			return nil, err
		}
		base := path.Base(parsed.Path)

		artifactResult.ArtifactURI = fmt.Sprintf("%s/%s/artifacts/%s", artifactLocation, runID, base)
		artifactResult.ArtifactURL = a.artifactURL(runID, experimentID)
		return &artifactResult, nil
	}
	glog.Warningf("Encountered unsupported artifact type: %v", schemaTitle)
	return nil, nil
}

func (a *ArtifactProvider) NestedRunsSupported() bool {
	return true
}

func (a *ArtifactProvider) metricURL(runId, experimentID string) string {
	uri := fmt.Sprintf("%s/#/experiments/%s/runs/%s/model-metrics", a.client.baseHost, experimentID, runId)
	return uri
}

func (a *ArtifactProvider) artifactURL(runId, experimentID string) string {
	uri := fmt.Sprintf("%s/#/experiments/%s/runs/%s/artifacts", a.client.baseHost, experimentID, runId)
	return uri
}
