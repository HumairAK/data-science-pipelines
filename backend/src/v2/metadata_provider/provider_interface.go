package metadata_provider

import (
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

type ProviderExperiment struct {
	ID          string
	Name        string
	Description string
	Namespace   string
}

type ProviderRun struct {
	ID          string
	Name        string
	Description string
	Status      string
}

type ArtifactResult struct {
	artifactURI string
	artifactURL string
}

type RunParameter struct {
	name  string
	value string
}

type Validator interface {
	ValidateRun(kfpRun *api.CreateRunRequest) error
	ValidateExperiment(experiment *api.CreateExperimentRequest) error
}

type RunProvider interface {
	GetRun(experimentID string, kfpRunID string) (*ProviderRun, error)
	CreateRun(
		experimentID string,
		kfpRun model.Run,
		parameters []RunParameter,
		parentRunID string,
	) (*ProviderRun, error)
	UpdateRunStatus(
		experimentID string,
		kfpRunID string,
		kfpRunStatus model.RuntimeState,
	) error
}

type MetadataArtifactProvider interface {
	LogOutputArtifact(
		experimentID string,
		runID string,
		runtimeArtifact *pipelinespec.RuntimeArtifact,
		defaultArtifactURI string) (ArtifactResult, error)
}
