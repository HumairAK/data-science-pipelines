package metadata_v2

import (
	"context"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
)

// For the ability to allow the user to use the mlflow in an auth safe way we can
// follow the following arch:
// We create a temporary user/password in mlflow and assign it permission for the experiment
// to which that run belongs, and then use those creds in the launcher's environment
// as env vars

// launcher will need
type ProviderExperiment struct {
	ID             string
	Name           string
	Description    string
	Namespace      string
	CreationTime   int64
	LastUpdateTime int64
}

// ProviderRuns map to Root, Dag, and Container driver executions.
type ProviderRun struct {
	ID           string
	Name         string
	Description  string
	Status       string
	CreationTime int64
	EndTime      int64
}

// Provide un opinionated way to provide artifact location to MLFLow when creating an
// experiment via passthrough API. And, if not provided then:
// For experiments, we use default bucket path

// MetadataExperimentProvider
// API Server creates/manages experiments
// Anything that requires Api server to manage storage should
// be scoped to a separate interface utilized only by API Server
type MetadataExperimentProvider interface {
	// GetExperimentStore should return a provider implementation of ExperimentStoreInterface
	// Experiment store will need to check to see if the artifact_location (e.g. in mlflow case)
	// is using a bucket in s3 (or something else entirely) that is not configured or supported
	// by this kfp deployment
	GetExperimentStore() storage.ExperimentStoreInterface
	// GetProviderConfig
	// ExperimentStore createExperiment will need to use a passthrough providerConfig object
	// createExperiment should also ValidateExperiment()
	GetProviderConfig() ProviderConfig
}

type ProviderConfig map[string]interface{}

// MetadataProviderValidator provides static validation utilities
type MetadataProviderValidator interface {
	ValidateRun(kfpRun api.CreateRunRequest) error
	ValidateExperiment(experiment api.CreateExperimentRequest) error
}

// MetadataRunProvider
// Driver/Launcher use the below operations
// Note: probably don't use api.Run below, model.Run is possibly better, or other alternatives
type MetadataRunProvider interface {
	// GetRun fetches the provider run mapped to this kfpRunID.
	GetRun(experimentID string, kfpRunID string) (*ProviderRun, error)
	// CreateRun create KFP run as a Provider Run.
	// Provider decides how/what from KFPRun gets stored as metadata into the provider run system.
	CreateRun(
		experimentID string,
		kfpRun api.Run,
		parameters []map[string]string,
	) (*ProviderRun, error)
	// LinkParentChildRuns is used for nesting Container and Dag provider runs under a Root provider run.
	LinkParentChildRuns(parentProviderRunID string, childProviderRunID string) error
	UpdateRunStatus(experimentID string, kfpRunID string, kfpRunStatus model.RuntimeState) error
}

type ArtifactResult struct {
	artifactURI string
	artifactURL string
}
type RunParameter struct {
	name  string
	value string
}

type ProviderArtifact interface {
	// Called in the launcher, around the sametime as record artifact
	// RuntimeArtifact implements:
	// GetMetrics()
	// GetModel()
	// GetDataset()

	// Provider can optionally use a defaultArtifactURI if they want to respect
	// KFP's Artifact URI format, or they can override and return the newly formatted URI
	// In the MLFlow case, artifact organization in the store bucket is very opinionated
	// We cannot set artifact path for a run level. So we offload URI creation to the provider.
	logOutputArtifact(
		experimentID string,
		runID string,
		runtimeArtifact pipelinespec.RuntimeArtifact,
		defaultArtifactURI string) (ArtifactResult, error)

	// Consider optionally
	listArtifact(
		experimentID string,
		runID string,
	) ([]pipelinespec.RuntimeArtifact, error)
	listArtifacts(
		experimentID string,
		runID string,
	) ([]pipelinespec.RuntimeArtifact, error)
}

// ##################################################################################################################
// OLD
// ##################################################################################################################
type Experiment struct {
	ID          string
	Name        string
	Description string
	Namespace   string
	// timestamps
}
type MetadataInterfaceClient interface {
	CreatePipelineRun(ctx context.Context, runName, pipelineName, namespace, runResource, pipelineRoot, storeSessionInfo, experimentID string, mlmdParentRunID, mlmdExecutionID *int64) (*metadata.Pipeline, error)
	UpdatePipelineStatus(ctx context.Context, mlmdExecutionID *int64, status pb.Execution_State, experimentID string) error
	CreateExperiment(ctx context.Context, experimentName, experimentDescription, kfpExperimentID string) (*string, error)
	GetExperiment(ctx context.Context, kfpExperimentID string) (*Experiment, error)
	// LogRunMetric returns Metric URI
	LogRunMetric(ctx context.Context, experimentID string, mlmdExecutionID int64, metricName string, metricValue float64) (*string, error)
	LogParameter(ctx context.Context, experimentID string, mlmdExecutionID int64, parameterName string, parameterValue string) error
	//LogInput(ctx context.Context, experimentID string, mlmdExecutionID int64, name, digest, sourceType, source, schema, profile string) error
	//LogModel(ctx context.Context, experimentID string, mlmdExecutionID int64, parameterName string, parameterValue string) error
}
