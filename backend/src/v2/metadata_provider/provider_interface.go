package metadata_provider

import (
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

type ProviderExperiment struct {
	ID             string
	Name           string
	Description    string
	Namespace      string
	CreationTime   int64
	LastUpdateTime int64
}

type ProviderRun struct {
	ID           string
	Name         string
	Description  string
	Status       string
	CreationTime int64
	EndTime      int64
}

type ArtifactResult struct {
	artifactURI string
	artifactURL string
}

type RunParameter struct {
	name  string
	value string
}
type ProviderConfig map[string]interface{}

type MetadataProviderValidator interface {
	ValidateRun(kfpRun *api.CreateRunRequest) error
	ValidateExperiment(experiment *api.CreateExperimentRequest) error
}

type MetadataRunProvider interface {
	GetRun(experimentID string, kfpRunID string) (*ProviderRun, error)
	CreateRun(
		experimentID string,
		kfpRun model.Run,
		parameters []RunParameter,
	) (*ProviderRun, error)
	LinkParentChildRuns(
		parentProviderRunID string,
		childProviderRunID string,
	) error
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
		runtimeArtifact pipelinespec.RuntimeArtifact,
		defaultArtifactURI string) (ArtifactResult, error)
}
