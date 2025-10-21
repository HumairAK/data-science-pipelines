package common

import (
	"context"
	"fmt"

	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"google.golang.org/protobuf/types/known/structpb"
)

// DriverAPI is a minimal interface exposing just the operations the drivers need
// from the KFP API server. It abstracts over RunService and ArtifactService.
//
// This indirection lets us unit test drivers and also evolve the underlying
// client without touching driver logic.
//
// Note: We intentionally do not expose the full apiclient.Client here.
// Only the small surface area needed by drivers is included.

type DriverAPI interface {
	// Run operations
	GetRun(ctx context.Context, req *gc.GetRunRequest) (*gc.Run, error)

	// Task operations
	CreateTask(ctx context.Context, req *gc.CreateTaskRequest) (*gc.PipelineTaskDetail, error)
	UpdateTask(ctx context.Context, req *gc.UpdateTaskRequest) (*gc.PipelineTaskDetail, error)
	GetTask(ctx context.Context, req *gc.GetTaskRequest) (*gc.PipelineTaskDetail, error)
	ListTasks(ctx context.Context, req *gc.ListTasksRequest) (*gc.ListTasksResponse, error)

	// Artifact operations
	CreateArtifact(ctx context.Context, req *gc.CreateArtifactRequest) (*gc.Artifact, error)
	ListArtifactsByURI(ctx context.Context, uri, namespace string) ([]*gc.Artifact, error)
	ListArtifactTasks(ctx context.Context, req *gc.ListArtifactTasksRequest) (*gc.ListArtifactTasksResponse, error)
	CreateArtifactTask(ctx context.Context, req *gc.CreateArtifactTaskRequest) (*gc.ArtifactTask, error)
	CreateArtifactTasks(ctx context.Context, req *gc.CreateArtifactTasksBulkRequest) (*gc.CreateArtifactTasksBulkResponse, error)

	// Pipeline version operations
	GetPipelineVersion(ctx context.Context, req *gc.GetPipelineVersionRequest) (*gc.PipelineVersion, error)
	FetchPipelineSpecFromRun(ctx context.Context, pipelineSpecStruct *structpb.Struct, run *gc.Run) (*structpb.Struct, error)
}

// kfpAPI adapts apiclient.Client to DriverAPI.
// It is a thin wrapper delegating to the generated gRPC clients.

type kfpAPI struct {
	c *apiclient.Client
}

// NewDriverAPI wraps the apiclient.Client into a DriverAPI.
func NewDriverAPI(c *apiclient.Client) DriverAPI {
	return &kfpAPI{c: c}
}

// Implement DriverAPI by forwarding calls to typed clients.

func (k *kfpAPI) GetRun(ctx context.Context, req *gc.GetRunRequest) (*gc.Run, error) {
	return k.c.Run.GetRun(ctx, req)
}

func (k *kfpAPI) CreateTask(ctx context.Context, req *gc.CreateTaskRequest) (*gc.PipelineTaskDetail, error) {
	return k.c.Run.CreateTask(ctx, req)
}

func (k *kfpAPI) UpdateTask(ctx context.Context, req *gc.UpdateTaskRequest) (*gc.PipelineTaskDetail, error) {
	return k.c.Run.UpdateTask(ctx, req)
}

func (k *kfpAPI) GetTask(ctx context.Context, req *gc.GetTaskRequest) (*gc.PipelineTaskDetail, error) {
	return k.c.Run.GetTask(ctx, req)
}

func (k *kfpAPI) ListTasks(ctx context.Context, req *gc.ListTasksRequest) (*gc.ListTasksResponse, error) {
	return k.c.Run.ListTasks(ctx, req)
}

func (k *kfpAPI) CreateArtifact(ctx context.Context, req *gc.CreateArtifactRequest) (*gc.Artifact, error) {
	return k.c.Artifact.CreateArtifact(ctx, req)
}

func (k *kfpAPI) ListArtifactsByURI(ctx context.Context, uri, namespace string) ([]*gc.Artifact, error) {
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

func (k *kfpAPI) ListArtifactTasks(ctx context.Context, req *gc.ListArtifactTasksRequest) (*gc.ListArtifactTasksResponse, error) {
	return k.c.Artifact.ListArtifactTasks(ctx, req)
}

func (k *kfpAPI) CreateArtifactTask(ctx context.Context, req *gc.CreateArtifactTaskRequest) (*gc.ArtifactTask, error) {
	return k.c.Artifact.CreateArtifactTask(ctx, req)
}

func (k *kfpAPI) CreateArtifactTasks(ctx context.Context, req *gc.CreateArtifactTasksBulkRequest) (*gc.CreateArtifactTasksBulkResponse, error) {
	return k.c.Artifact.CreateArtifactTasksBulk(ctx, req)
}

func (k *kfpAPI) GetPipelineVersion(ctx context.Context, req *gc.GetPipelineVersionRequest) (*gc.PipelineVersion, error) {
	return k.c.Pipeline.GetPipelineVersion(ctx, req)
}

func (k *kfpAPI) FetchPipelineSpecFromRun(ctx context.Context, pipelineSpecStruct *structpb.Struct, run *gc.Run) (*structpb.Struct, error) {
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
	return pipelineSpecStruct, nil
}

func (k *kfpAPI) FindMatchedArtifact(artifact *gc.Artifact) (*gc.Artifact, error) {

	return nil, nil
}
