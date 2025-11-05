package kfpapi

import (
	"context"
	"fmt"

	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"google.golang.org/protobuf/types/known/structpb"
)

// API is a minimal interface exposing KFP API operations needed by drivers and launchers.
// It abstracts over RunService, ArtifactService, and PipelineService.
//
// This indirection lets us unit test components and also evolve the underlying
// client without touching driver/launcher logic.
//
// Note: We intentionally do not expose the full apiclient.Client here.
// Only the small surface area needed is included.

type API interface {
	// Run operations
	GetRun(ctx context.Context, req *gc.GetRunRequest) (*gc.Run, error)

	// Task operations
	CreateTask(ctx context.Context, req *gc.CreateTaskRequest) (*gc.PipelineTaskDetail, error)
	UpdateTask(ctx context.Context, req *gc.UpdateTaskRequest) (*gc.PipelineTaskDetail, error)
	UpdateTasksBulk(ctx context.Context, req *gc.UpdateTasksBulkRequest) (*gc.UpdateTasksBulkResponse, error)
	GetTask(ctx context.Context, req *gc.GetTaskRequest) (*gc.PipelineTaskDetail, error)
	ListTasks(ctx context.Context, req *gc.ListTasksRequest) (*gc.ListTasksResponse, error)

	// Artifact operations
	CreateArtifact(ctx context.Context, req *gc.CreateArtifactRequest) (*gc.Artifact, error)
	CreateArtifactsBulk(ctx context.Context, req *gc.CreateArtifactsBulkRequest) (*gc.CreateArtifactsBulkResponse, error)
	ListArtifactsByURI(ctx context.Context, uri, namespace string) ([]*gc.Artifact, error)
	ListArtifactTasks(ctx context.Context, req *gc.ListArtifactTasksRequest) (*gc.ListArtifactTasksResponse, error)
	CreateArtifactTask(ctx context.Context, req *gc.CreateArtifactTaskRequest) (*gc.ArtifactTask, error)
	CreateArtifactTasks(ctx context.Context, req *gc.CreateArtifactTasksBulkRequest) (*gc.CreateArtifactTasksBulkResponse, error)

	// Pipeline version operations
	GetPipelineVersion(ctx context.Context, req *gc.GetPipelineVersionRequest) (*gc.PipelineVersion, error)
	FetchPipelineSpecFromRun(ctx context.Context, pipelineSpecStruct *structpb.Struct, run *gc.Run) (*structpb.Struct, error)
}

// clientAdapter adapts apiclient.Client to API.
// It is a thin wrapper delegating to the generated gRPC clients.

type clientAdapter struct {
	c *apiclient.Client
}

// New wraps the apiclient.Client into an API interface.
func New(c *apiclient.Client) API {
	return &clientAdapter{c: c}
}

// Implement API by forwarding calls to typed clients.

func (k *clientAdapter) GetRun(ctx context.Context, req *gc.GetRunRequest) (*gc.Run, error) {
	return k.c.Run.GetRun(ctx, req)
}

func (k *clientAdapter) CreateTask(ctx context.Context, req *gc.CreateTaskRequest) (*gc.PipelineTaskDetail, error) {
	return k.c.Run.CreateTask(ctx, req)
}

func (k *clientAdapter) UpdateTask(ctx context.Context, req *gc.UpdateTaskRequest) (*gc.PipelineTaskDetail, error) {
	return k.c.Run.UpdateTask(ctx, req)
}

func (k *clientAdapter) UpdateTasksBulk(ctx context.Context, req *gc.UpdateTasksBulkRequest) (*gc.UpdateTasksBulkResponse, error) {
	return k.c.Run.UpdateTasksBulk(ctx, req)
}

func (k *clientAdapter) GetTask(ctx context.Context, req *gc.GetTaskRequest) (*gc.PipelineTaskDetail, error) {
	return k.c.Run.GetTask(ctx, req)
}

func (k *clientAdapter) ListTasks(ctx context.Context, req *gc.ListTasksRequest) (*gc.ListTasksResponse, error) {
	return k.c.Run.ListTasks(ctx, req)
}

func (k *clientAdapter) CreateArtifact(ctx context.Context, req *gc.CreateArtifactRequest) (*gc.Artifact, error) {
	return k.c.Artifact.CreateArtifact(ctx, req)
}

func (k *clientAdapter) CreateArtifactsBulk(ctx context.Context, req *gc.CreateArtifactsBulkRequest) (*gc.CreateArtifactsBulkResponse, error) {
	return k.c.Artifact.CreateArtifactsBulk(ctx, req)
}

func (k *clientAdapter) ListArtifactsByURI(ctx context.Context, uri, namespace string) ([]*gc.Artifact, error) {
	filter := &gc.Filter{
		Predicates: []*gc.Predicate{
			{Key: "uri", Operation: gc.Predicate_EQUALS, Value: &gc.Predicate_StringValue{StringValue: uri}},
		}}

	const pageSize = 100
	var allArtifacts []*gc.Artifact
	nextPageToken := ""

	for {
		artifactsResponse, err := k.c.Artifact.ListArtifacts(ctx, &gc.ListArtifactRequest{
			Namespace: namespace,
			Filter:    filter.String(),
			PageSize:  pageSize,
			PageToken: nextPageToken,
		})
		if err != nil {
			return nil, err
		}

		allArtifacts = append(allArtifacts, artifactsResponse.GetArtifacts()...)
		nextPageToken = artifactsResponse.GetNextPageToken()

		if nextPageToken == "" {
			break
		}
	}

	return allArtifacts, nil
}

func (k *clientAdapter) ListArtifactTasks(ctx context.Context, req *gc.ListArtifactTasksRequest) (*gc.ListArtifactTasksResponse, error) {
	return k.c.Artifact.ListArtifactTasks(ctx, req)
}

func (k *clientAdapter) CreateArtifactTask(ctx context.Context, req *gc.CreateArtifactTaskRequest) (*gc.ArtifactTask, error) {
	return k.c.Artifact.CreateArtifactTask(ctx, req)
}

func (k *clientAdapter) CreateArtifactTasks(ctx context.Context, req *gc.CreateArtifactTasksBulkRequest) (*gc.CreateArtifactTasksBulkResponse, error) {
	return k.c.Artifact.CreateArtifactTasksBulk(ctx, req)
}

func (k *clientAdapter) GetPipelineVersion(ctx context.Context, req *gc.GetPipelineVersionRequest) (*gc.PipelineVersion, error) {
	return k.c.Pipeline.GetPipelineVersion(ctx, req)
}

func (k *clientAdapter) FetchPipelineSpecFromRun(ctx context.Context, pipelineSpecStruct *structpb.Struct, run *gc.Run) (*structpb.Struct, error) {
	if run.GetPipelineSpec() != nil {
		pipelineSpecStruct = run.GetPipelineSpec()
	} else if run.GetPipelineVersionReference() != nil {
		pvr := run.GetPipelineVersionReference()
		pipeline, err := k.GetPipelineVersion(ctx, &gc.GetPipelineVersionRequest{
			PipelineId:        pvr.GetPipelineId(),
			PipelineVersionId: pvr.GetPipelineVersionId(),
		})
		if err != nil {
			return nil, err
		}
		pipelineSpecStruct = pipeline.GetPipelineSpec()
	} else {
		return nil, fmt.Errorf("pipeline spec is not set")
	}
	if len(pipelineSpecStruct.GetFields()) > 1 {
		return pipelineSpecStruct.GetFields()["pipeline_spec"].GetStructValue(), nil
	}
	return pipelineSpecStruct, nil
}
