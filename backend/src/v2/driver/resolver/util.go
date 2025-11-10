package resolver

import (
	"fmt"

	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
)

func findParameterByIOKey(
	key string,
	pms []ParameterMetadata,
) (*apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter, error) {
	for _, pm := range pms {
		if pm.ParameterIO.GetParameterKey() == key {
			return pm.ParameterIO, nil
		}
	}
	return nil, fmt.Errorf("parameter not found")
}

func findArtifactByIOKey(
	key string,
	ams []ArtifactMetadata,
) (*apiV2beta1.PipelineTaskDetail_InputOutputs_IOArtifact, error) {
	for _, am := range ams {
		if am.Key == key {
			return am.ArtifactIO, nil
		}
	}
	return nil, fmt.Errorf("artifact not found")
}
