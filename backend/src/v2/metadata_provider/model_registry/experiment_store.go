package model_registry

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/kubeflow/model-registry/pkg/openapi"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	commonutils "github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/config"
)

// Ensure ModelRegistryExperimentProvider implements MetadataExperimentProvider
var _ storage.ExperimentStoreInterface = &ExperimentStore{}

// ExperimentStore
// Used by api server creates/manages experiments
type ExperimentStore struct {
	client *Client
}

func addTag(tags *map[string]openapi.MetadataValue, key string, value string) {
	mv := openapi.NewMetadataStringValueWithDefaults()
	mv.SetStringValue(value)
	(*tags)[key] = openapi.MetadataStringValueAsMetadataValue(mv)
}

func (s *ExperimentStore) CreateExperiment(baseExperiment *model.Experiment, providerConfig config.GenericProviderConfig) (*model.Experiment, error) {

	mv := openapi.NewMetadataStringValueWithDefaults()
	mv.SetStringValue(baseExperiment.Description)
	var tags *map[string]openapi.MetadataValue

	namespace := baseExperiment.Namespace
	// Experiments in ModelRegistry have no notion of namespaces
	// So we label experiments using tags, but tags are not
	// unique, thus we must also include the namespace in the name
	if namespace != "" {
		addTag(tags, NameTag, baseExperiment.Name)
		addTag(tags, NamespaceTag, baseExperiment.Namespace)
		baseExperiment.Name = BuildExperimentNamespaceName(baseExperiment.Name, namespace)
	}

	var artifactLocation *string
	if providerConfig != nil {
		creationConfig, err := ConvertToExperimentCreationConfig(providerConfig)
		if err != nil {
			return nil, err
		}
		artifactLocation = &creationConfig.ArtifactLocation
	}

	experiment, err := s.client.createExperiment(
		baseExperiment.Name,
		baseExperiment.Description,
		tags,
		artifactLocation)
	if err != nil {
		return nil, err
	}

	modelExperiment, err := mrExperimentToModelExperiment(*experiment)
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
	experimentModel, err := mrExperimentToModelExperiment(*experiment)
	if err != nil {
		return nil, err
	}
	return experimentModel, nil
}

// GetExperimentByNameNamespace
// ToDO: handle namespace case for multi user mode when model registry
// adds support for searching by custom properties, or via namespace field
func (s *ExperimentStore) GetExperimentByNameNamespace(name string, namespace string) (*model.Experiment, error) {
	experiment, err := s.client.getExperimentByName(name)
	if err != nil {
		return nil, err
	}
	return mrExperimentToModelExperiment(*experiment)
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

	// TODO add sorting logic, sorting info is in opts.token and opts.sortby
	if namespace != "" {
		// ToDO: handle namespace case for multi user mode when model registry
		// adds support for searching by custom properties, or via namespace field
	}

	viewType := openapi.ExperimentState(determineStorageState(opts.Filter))
	experiments, nextPageToken, err := s.client.getExperiments(
		int64(opts.PageSize),
		opts.PageToken,
		"",
		"",
		&viewType,
	)
	if err != nil {
		return errorF(err)
	}
	for _, experiment := range experiments {
		experimentModel, conversionErr := mrExperimentToModelExperiment(experiment)
		if conversionErr != nil {
			return errorF(conversionErr)
		}
		experimentModels = append(experimentModels, experimentModel)
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
	return fmt.Errorf("Permanently deleting experiments is not supported in ModelRegistry. Please use the archive API instead.")
}

// SetLastRunTimestamp
// Don't fail to avoid blocking pipeline creation
func (s *ExperimentStore) SetLastRunTimestamp(run *model.Run) error {
	glog.Warning("SetLastRunTimestamp is not implemented for ModelRegistry.")
	return nil
}

func determineStorageState(f *filter.Filter) string {
	if f == nil {
		return ""
	}
	if match := f.FilterOn("experiments.StorageState", string(model.StorageStateAvailable), apiv2beta1.Predicate_EQUALS); match != nil && *match {
		return string(openapi.EXPERIMENTSTATE_LIVE)
	}
	if match := f.FilterOn("experiments.StorageState", string(model.StorageStateAvailable), apiv2beta1.Predicate_NOT_EQUALS); match != nil && *match {
		return string(openapi.EXPERIMENTSTATE_ARCHIVED)
	}
	if match := f.FilterOn("experiments.StorageState", string(model.StorageStateArchived), apiv2beta1.Predicate_EQUALS); match != nil && *match {
		return string(openapi.EXPERIMENTSTATE_ARCHIVED)
	}
	if match := f.FilterOn("experiments.StorageState", string(model.StorageStateArchived), apiv2beta1.Predicate_NOT_EQUALS); match != nil && *match {
		return string(openapi.EXPERIMENTSTATE_LIVE)
	}
	return ""
}
