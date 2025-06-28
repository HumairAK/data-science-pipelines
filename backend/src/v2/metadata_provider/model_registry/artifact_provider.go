package model_registry

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
		err := a.client.logBatch(runID, metrics, []types.Param{}, nil)
		if err != nil {
			return nil, err
		}
		// Since there are multiple metrics, there is no way to tie multie URIs
		// to a single KFP artifact, so we just return the Metrics path for MLFlow UI
		metricURL := a.metricURL(runID, experimentID)
		artifactResult.ArtifactURL = metricURL
		return &artifactResult, nil
	case util.SchemaTitleArtifact:
		artifactLocation := a.client.DefaultArtifactTrackingURI

		// TODO: Not sure if this will work with MR plugin, see sibling code in CreateExperiment()
		//experiment, err := a.client.getExperiment(experimentID)
		//if err != nil {
		//	return nil, err
		//}
		//defaultURI := experiment.GetCustomProperties()[ExperimentArtifactURI]
		//if defaultURI.MetadataStringValue != nil && defaultURI.MetadataStringValue.GetStringValue() != "" {
		//	artifactLocation = defaultURI.MetadataStringValue.GetStringValue()
		//}

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

func (a *ArtifactProvider) metricURL(runId, experimentID string) string {
	uri := fmt.Sprintf("%s/#/experiments/%s/runs/%s/model-metrics", a.client.Host, experimentID, runId)
	return uri
}

func (a *ArtifactProvider) artifactURL(runId, experimentID string) string {
	uri := fmt.Sprintf("%s/#/experiments/%s/runs/%s/artifacts", a.client.Host, experimentID, runId)
	return uri
}
