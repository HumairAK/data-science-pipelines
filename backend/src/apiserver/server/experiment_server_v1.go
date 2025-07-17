package server

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
)

type ExperimentServerV1 struct {
	*BaseExperimentServer
	apiv1beta1.UnimplementedExperimentServiceServer
}

func (s *ExperimentServerV1) GetExperimentV1(ctx context.Context, request *apiv1beta1.GetExperimentRequest) (
	*apiv1beta1.Experiment, error,
) {
	if s.options.CollectMetrics {
		getExperimentRequests.Inc()
	}

	experiment, err := s.getExperiment(ctx, request.GetId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to fetch v1beta1 experiment")
	}

	apiExperiment := toApiExperimentV1(experiment)
	if apiExperiment == nil {
		return nil, util.NewInternalServerError(errors.New("Failed to convert internal experiment representation to its v1beta1 API counterpart"), "Failed to fetch v1beta1 experiment")
	}
	return apiExperiment, nil
}

func (s *ExperimentServerV1) ListExperimentsV1(ctx context.Context, request *apiv1beta1.ListExperimentsRequest) (
	*apiv1beta1.ListExperimentsResponse, error,
) {
	if s.options.CollectMetrics {
		listExperimentsV1Requests.Inc()
	}

	filterContext, err := validateFilterV1(request.ResourceReferenceKey)
	if err != nil {
		return nil, util.Wrap(err, "Validating v1beta1 filter failed")
	}
	namespace := ""
	if filterContext.ReferenceKey != nil {
		if filterContext.ReferenceKey.Type == model.NamespaceResourceType {
			namespace = filterContext.ReferenceKey.ID
		} else {
			return nil, util.NewInvalidInputError("Failed to list v1beta1 experiment due to invalid resource reference key. It must be of type 'Namespace' and contain an existing or empty namespace, but you provided %v of type %v", filterContext.ReferenceKey.ID, filterContext.ReferenceKey.Type)
		}
	}

	opts, err := validatedListOptions(&model.Experiment{}, request.GetPageToken(), int(request.GetPageSize()), request.GetSortBy(), request.GetFilter(), "v1beta1")
	if err != nil {
		return nil, util.Wrap(err, "Failed to create list options")
	}

	experiments, totalSize, nextPageToken, err := s.listExperiments(
		ctx,
		request.GetPageToken(),
		request.GetPageSize(),
		request.GetSortBy(),
		opts,
		namespace,
	)
	if err != nil {
		return nil, util.Wrap(err, "List v1beta1 experiments failed")
	}
	return &apiv1beta1.ListExperimentsResponse{
		Experiments:   toApiExperimentsV1(experiments),
		TotalSize:     totalSize,
		NextPageToken: nextPageToken,
	}, nil
}

func (s *ExperimentServerV1) DeleteExperimentV1(ctx context.Context, request *apiv1beta1.DeleteExperimentRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		deleteExperimentRequests.Inc()
	}

	if err := s.deleteExperiment(ctx, request.GetId()); err != nil {
		return nil, util.Wrap(err, "Failed to delete v1beta1 experiment")
	}

	if s.options.CollectMetrics {
		experimentCount.Dec()
	}
	return &empty.Empty{}, nil
}

func (s *ExperimentServerV1) ArchiveExperimentV1(ctx context.Context, request *apiv1beta1.ArchiveExperimentRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		archiveExperimentRequests.Inc()
	}
	if err := s.archiveExperiment(ctx, request.GetId()); err != nil {
		return nil, util.Wrap(err, "Failed to archive v1beta1 experiment")
	}
	return &empty.Empty{}, nil
}

func (s *ExperimentServerV1) UnarchiveExperimentV1(ctx context.Context, request *apiv1beta1.UnarchiveExperimentRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		unarchiveExperimentRequests.Inc()
	}

	if err := s.unarchiveExperiment(ctx, request.GetId()); err != nil {
		return nil, util.Wrap(err, "Failed to unarchive v1beta1 experiment")
	}
	return &empty.Empty{}, nil
}

func (s *ExperimentServerV1) CreateExperimentV1(ctx context.Context, request *apiv1beta1.CreateExperimentRequest) (
	*apiv1beta1.Experiment, error,
) {
	if s.options.CollectMetrics {
		createExperimentRequests.Inc()
	}

	modelExperiment, err := toModelExperiment(request.GetExperiment())
	if err != nil {
		return nil, util.Wrap(err, "[ExperimentServer]: Failed to create a v1beta1 experiment due to conversion error")
	}

	newExperiment, err := s.createExperiment(ctx, modelExperiment)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a v1beta1 experiment")
	}

	if s.options.CollectMetrics {
		experimentCount.Inc()
	}

	apiExperiment := toApiExperimentV1(newExperiment)
	if apiExperiment == nil {
		return nil, util.NewInternalServerError(errors.New("Failed to convert internal experiment representation to its API counterpart"), "Failed to create v1beta1 experiment")
	}
	return apiExperiment, nil
}

func NewExperimentServerV1(base *BaseExperimentServer) *ExperimentServerV1 {
	return &ExperimentServerV1{
		BaseExperimentServer: base,
	}
}
