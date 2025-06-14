package metadata_provider

import (
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	corev1 "k8s.io/api/core/v1"
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

// Validator implements callback validation methods
type Validator interface {
	// ValidateRun will be called when a KFP run is created
	ValidateRun(kfpRun *api.CreateRunRequest) error
	// ValidateExperiment will be called when an experiment is created
	ValidateExperiment(experiment *api.CreateExperimentRequest) error
	// ValidateConfig will be called on KFP start up when the pipeline config.json is parsed.
	ValidateConfig(config common.UnstructuredJSON, envvars []corev1.EnvVar) error
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

type ProviderFactory interface {
	NewExperimentStore(cfg common.UnstructuredJSON) (storage.ExperimentStoreInterface, error)
	NewRunProvider(cfg common.UnstructuredJSON) (RunProvider, error)
	NewMetadataArtifactProvider(cfg common.UnstructuredJSON) (MetadataArtifactProvider, error)
	NewValidator(cfg common.UnstructuredJSON) (Validator, error)
}
