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
	// ValidateConfig will be called on KFP, Driver, & Launcher startup when the pipeline config.json is parsed.
	ValidateConfig(config config.GenericProviderConfig) error
}

type RunProvider interface {
	GetRun(experimentID string, ProviderRunID string) (*ProviderRun, error)
	CreateRun(
		experimentID string,
		// TODO: replace kfprun and taskname with apiv2beta1.TaskDetail
		kfpRun *apiv2beta1.Run,
		ProviderRunName string,
		parameters []RunParameter,
		parentRunID string,
	) (*ProviderRun, error)
	UpdateRunStatus(
		providerRunID string,
		kfpRunStatus model.RuntimeState,
	) error

	// ExecutorPatch returns a Pod Patch that will be merged with the executor pod.
	// Return nil patch with nil error if no patch is needed.
	ExecutorPatch(experimentID string, providerRunID string) (*corev1.PodSpec, error)
}

type MetadataArtifactProvider interface {
	// LogOutputArtifact will be called when a KFP artifact is logged.
	// * If the artifact is not supported, return nil runtimeArtifact and nil error.
	// * If the artifact is supported, return the artifact result and nil error.
	// * If the artifact logging is supported, but doesn't require to be written out
	//   to object store (e.g. MLFlow metrics), then return empty ArtifactResult.ArtifactURI.
	LogOutputArtifact(
		runID string,
		experimentID string,
		runtimeArtifact *pipelinespec.RuntimeArtifact,
	) (*ArtifactResult, error)
}

type ProviderFactory interface {
	NewExperimentStore(cfg config.GenericProviderConfig) (storage.ExperimentStoreInterface, error)
	NewRunProvider(cfg config.GenericProviderConfig) (RunProvider, error)
	NewMetadataArtifactProvider(cfg config.GenericProviderConfig) (MetadataArtifactProvider, error)
	NewValidator(cfg config.GenericProviderConfig) (Validator, error)
}
