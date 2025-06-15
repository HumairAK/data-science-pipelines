package metadata_provider

import (
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/config"
	corev1 "k8s.io/api/core/v1"
)

type ProviderExperiment struct {
	ID          string
	Name        string
	Description string
	Namespace   string
}

type ProviderRun struct {
	ID     string
	Name   string
	Status string
}

type ArtifactResult struct {
	Name        string
	ArtifactURI string
	ArtifactURL string
}

type RunParameter struct {
	Name  string
	Value string
}

// Validator implements callback validation methods
type Validator interface {
	// ValidateRun will be called when a KFP run is created
	ValidateRun(kfpRun *api.CreateRunRequest) error
	// ValidateExperiment will be called when an experiment is created
	ValidateExperiment(experiment *api.CreateExperimentRequest) error
	// ValidateConfig will be called on KFP start up when the pipeline config.json is parsed.
	ValidateConfig(config config.GenericProviderConfig, envvars []corev1.EnvVar) error
}

type RunProvider interface {
	GetRun(experimentID string, ProviderRunID string) (*ProviderRun, error)
	CreateRun(
		experimentID string,
	// TODO: replace kfprun and taskname with apiv2beta1.TaskDetails
		kfpRun *apiv2beta1.Run,
		ProviderRunName string,
		parameters []RunParameter,
		parentRunID string,
	) (*ProviderRun, error)
	UpdateRunStatus(
		providerRunID string,
		kfpRunStatus model.RuntimeState,
	) error

	// NestedRunsSupported
	// Enable run nesting by having this function return true
	// Otherwise all kfp pipeline run tasks are logged flatly
	// TODO(humairak): make this a top level config
	NestedRunsSupported() bool
}

type MetadataArtifactProvider interface {
	// LogOutputArtifact will be called when a KFP artifact is logged.
	// If the artifact is not supported, return nil runtimeArtifact and no error.
	// If the artifact is supported, return the artifact result and no error.
	LogOutputArtifact(
		runID string,
		runtimeArtifact *pipelinespec.RuntimeArtifact,
	) (*ArtifactResult, error)

	// NestedRunsSupported
	// If true then the metadata provider will log artifacts for parent runs.
	// TODO(humairak): make this a top level config
	NestedRunsSupported() bool
}

type ProviderFactory interface {
	NewExperimentStore(cfg config.GenericProviderConfig) (storage.ExperimentStoreInterface, error)
	NewRunProvider(cfg config.GenericProviderConfig) (RunProvider, error)
	NewMetadataArtifactProvider(cfg config.GenericProviderConfig) (MetadataArtifactProvider, error)
	NewValidator(cfg config.GenericProviderConfig) (Validator, error)
}
