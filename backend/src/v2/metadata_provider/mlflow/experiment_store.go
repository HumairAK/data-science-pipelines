package mlflow

import (
	"fmt"
	"github.com/golang/glog"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	commonutils "github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/mlflow/types"
)

// Ensure MLFlowExperimentProvider implements MetadataExperimentProvider
var _ storage.ExperimentStoreInterface = &ExperimentStore{}

// ExperimentStore
// Used by api server creates/manages experiments
// Experiment store will need to check to see if the artifact_location (e.g. in mlflow case)
// is using a bucket in s3 (or something else entirely) that is not configured or supported
// by this kfp deployment.
type ExperimentStore struct {
	client *Client
}

func NewExperimentStore(config common.UnstructuredJSON) (storage.ExperimentStoreInterface, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, err
	}
	return &ExperimentStore{client: client}, nil
}

// CreateExperiment accepts a providerConfig which is used for
// provider-specific creation options.
// For example, MLFlow allows you to set the artifact_location for all artifacts
// uploaded in any run for a given experiment. The user should be able to configure this.
// If the provider config is not provided, we would use the default bucket path
func (s *ExperimentStore) CreateExperiment(baseExperiment *model.Experiment, providerConfig common.UnstructuredJSON) (*model.Experiment, error) {
	experimentTags := []types.ExperimentTag{
		{
			Key:   ExperimentDescriptionTag,
			Value: baseExperiment.Description,
		},
	}
	namespace := baseExperiment.Namespace
	// Experiments in MLFlow have no notion of namespaces
	// So we label experiments using tags, but tags are not
	// unique, thus we must also include the namespace in the name
	if namespace != "" {
		experimentTags = append(experimentTags,
			// For easy searching and future parsing, we keep the canonical name as a tag
			types.ExperimentTag{
				Key:   NameTag,
				Value: baseExperiment.Name,
			},
			types.ExperimentTag{
				Key:   NamespaceTag,
				Value: baseExperiment.Namespace,
			},
		)
		baseExperiment.Name = BuildExperimentNamespaceName(baseExperiment.Name, namespace)
	}

	var artifactLocation string
	if providerConfig != nil {
		creationConfig, err := ConvertToExperimentCreationConfig(providerConfig)
		if err != nil {
			return nil, err
		}
		artifactLocation = creationConfig.ArtifactLocation
	}

	mlflowExperimentID, err := s.client.createExperiment(
		baseExperiment.Name,
		artifactLocation,
		experimentTags,
	)
	if err != nil {
		return nil, err
	}

	experiment, err := s.client.getExperiment(mlflowExperimentID)
	if err != nil {
		return nil, err
	}

	modelExperiment, err := mlflowExperimentToModelExperiment(*experiment)
	if err != nil {
		return nil, err
	}
	return modelExperiment, nil
}

func (s *ExperimentStore) GetExperiment(uuid string) (*model.Experiment, error) {
	experiment, err := s.client.getExperiment(uuid)
	if err != nil {
		return nil, err
	}
	experimentModel, err := mlflowExperimentToModelExperiment(*experiment)
	if err != nil {
		return nil, err
	}
	return experimentModel, nil
}

// GetExperimentByNameNamespace returns the experiment with the given name and namespace.
// If no experiment is found, it returns an error.
func (s *ExperimentStore) GetExperimentByNameNamespace(name string, namespace string) (*model.Experiment, error) {
	filter := fmt.Sprintf("name='%s'", name)
	// In multi-user mode namespace is set, in which case we only list experiments in the given namespace
	if namespace != "" {
		filter = fmt.Sprintf("%s' AND tag.%s='%s'", filter, NamespaceTag, namespace)
	}
	experiments, _, err := s.client.searchExperiments(1, "", filter, []string{}, "")
	if err != nil {
		return nil, err
	}
	if len(experiments) < 1 {
		return nil, fmt.Errorf("no experiment found with name %s and namespace %s", name, namespace)
	}
	experiment := experiments[1]
	experimentModel, err := mlflowExperimentToModelExperiment(experiment)
	if err != nil {
		return nil, err
	}
	return experimentModel, nil
}

func (s *ExperimentStore) ListExperiments(filterContext *model.FilterContext, opts *list.Options) ([]*model.Experiment, int, string, error) {
	errorF := func(err error) ([]*model.Experiment, int, string, error) {
		return nil, 0, "", commonutils.NewInternalServerError(err, "Failed to list experiments: %v", err)
	}

	var namespace string
	var experimentModels []*model.Experiment

	if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.NamespaceResourceType {
		namespace = filterContext.ReferenceKey.ID
	}

	var filter string
	// TODO add sorting logic, sorting info is in opts.token and opts.sortby
	if namespace != "" {
		filter = fmt.Sprintf("tag.%s='%s'", NamespaceTag, namespace)
	}

	viewType := determineStorageState(opts.Filter)
	experiments, nextPageToken, err := s.client.searchExperiments(int64(opts.PageSize), opts.PageToken, filter, []string{}, types.ViewType(viewType))
	if err != nil {
		return errorF(err)
	}
	for _, experiment := range experiments {
		// if experiment is valid convert to model.Experiment and append to experimentModels
		err1 := validateMLFlowExperiment(experiment, namespace)
		if err1 == nil {
			experimentModel, err2 := mlflowExperimentToModelExperiment(experiment)
			if err2 != nil {
				return errorF(err2)
			}
			experimentModels = append(experimentModels, experimentModel)
		}
	}
	return experimentModels, len(experimentModels), nextPageToken, nil
}

func (s *ExperimentStore) ArchiveExperiment(expId string) error {
	err := s.client.deleteExperiment(expId)
	if err != nil {
		return err
	}
	return nil
}

func (s *ExperimentStore) UnarchiveExperiment(expId string) error {
	err := s.client.restoreExperiment(expId)
	if err != nil {
		return err
	}
	return nil
}

func (s *ExperimentStore) DeleteExperiment(expId string) error {
	return fmt.Errorf("Permanently deleting experiments is not supported in MLFlow. Please use the archive API instead.")
}

// SetLastRunTimestamp
// Don't fail to avoid blocking pipeline creation
func (s *ExperimentStore) SetLastRunTimestamp(run *model.Run) error {
	glog.Warning("SetLastRunTimestamp is not implemented for MLFlow.")
	return nil
}

func determineStorageState(f *filter.Filter) string {
	if f == nil {
		return ""
	}
	if match := f.FilterOn("experiments.StorageState", string(model.StorageStateAvailable), apiv2beta1.Predicate_EQUALS); match != nil && *match {
		return "ACTIVE_ONLY"
	}
	if match := f.FilterOn("experiments.StorageState", string(model.StorageStateAvailable), apiv2beta1.Predicate_NOT_EQUALS); match != nil && *match {
		return "DELETED_ONLY"
	}
	if match := f.FilterOn("experiments.StorageState", string(model.StorageStateArchived), apiv2beta1.Predicate_EQUALS); match != nil && *match {
		return "DELETED_ONLY"
	}
	if match := f.FilterOn("experiments.StorageState", string(model.StorageStateArchived), apiv2beta1.Predicate_NOT_EQUALS); match != nil && *match {
		return "ACTIVE_ONLY"
	}
	return ""
}
