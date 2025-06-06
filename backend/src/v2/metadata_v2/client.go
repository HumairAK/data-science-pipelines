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

type ProviderExperiment struct {
	ID             string
	Name           string
	Description    string
	Namespace      string
	CreationTime   int64
	LastUpdateTime int64
}

// ProviderRun can be map to oneOf(Root, Dag, Container) execution
type ProviderRun struct {
	ID           string
	Name         string
	Description  string
	Status       string
	CreationTime int64
	EndTime      int64
}

// Come up with an unopinionated way to provide artifact location to MLFLow when creating an
// experiment via passthrough API. And, if not provided, then:
// For experiments, we use a default bucket path

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
	// CreateExperiment() will need to also accept a providerConfig which is used for
	// provider specific creation options.
	// For example, MLFlow allows you to set the artifact_location for all artifacts
	// uploaded in any run for a given experiment. The user should be able to configure this.
}

// ProviderConfig is a map of key value pairs that can be used to configure the provider
// The KFP Frontend will be provided a config schema that it can absorb and use to populate
// the Experiment Creation (or potentially Run creation) forms. This is passed along with
// the rest of the experiment create request object, and validated by backend, then
// it is passed to the Create[Experiment,Run] handler.
type ProviderConfig map[string]interface{}

// MetadataProviderValidator provides static validation utilities
// Note:  api.CreateExperimentRequest (and possibly api.CreateRunRequest) will also need a ProviderConfig struct field
type MetadataProviderValidator interface {
	ValidateRun(kfpRun api.CreateRunRequest) error
	ValidateExperiment(experiment api.CreateExperimentRequest) error
}

// MetadataRunProvider
// Used by Driver/Launcher
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
	// LinkParentChildRuns is used for nesting Container and Dag provider runs under Dag or Root provider runs.
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
	// Called in the launcher, likely around the sametime as recordArtifact()
	// RuntimeArtifact will need to implement:
	// GetMetrics()
	// GetModel()
	// GetDataset()

	// Provider can optionally use a defaultArtifactURI if they want to respect
	// KFP's Artifact URI format, or they can override and return the newly formatted URI
	// In the MLFlow case; artifact organization in the store bucket is very opinionated
	// We cannot set an artifact path for a run level. So we offload URI creation to the provider.
	logOutputArtifact(
		experimentID string,
		runID string,
		runtimeArtifact pipelinespec.RuntimeArtifact,
		defaultArtifactURI string) (ArtifactResult, error)

	// Consider: Get/Listing of artifacts
}

// Other notes and considerations:
// There is an ask for runs to be able to be moved to different experiments
// this would come from the provider side of things, e.g., you move runs in mlflow to a different experiment
// (once mlflow implements this feature)
// but the issue is how do we then know to update the run's experiment UID in kfp runs table?
// one solution is for each run get we validate the run's experiment provider UID is accurate, but this feels overkill

// For the ability to allow the user to use mlflow in an auth safe way within their components we can
// follow the following approach:
// We create a temporary user/password in mlflow and assign it permissions for the experiment
// to which that component run belongs, and then use those creds in the launcher's environment
// as environment variables. Once the run is complete, we clean up the ephemeral user.

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
