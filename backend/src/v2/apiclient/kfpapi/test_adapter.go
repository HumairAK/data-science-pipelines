package kfpapi

import (
	"context"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// TestRunServiceAdapter adapts our MockAPI to implement the full RunServiceClient interface
// This allows us to inject MockAPI into components that expect the gRPC client interface.
type TestRunServiceAdapter struct {
	api API
}

// NewTestRunServiceAdapter creates an adapter that wraps a MockAPI
func NewTestRunServiceAdapter(api API) *TestRunServiceAdapter {
	return &TestRunServiceAdapter{api: api}
}

// Implemented methods (delegate to MockAPI)

func (a *TestRunServiceAdapter) GetRun(ctx context.Context, req *apiv2beta1.GetRunRequest, opts ...grpc.CallOption) (*apiv2beta1.Run, error) {
	return a.api.GetRun(ctx, req)
}

func (a *TestRunServiceAdapter) CreateTask(ctx context.Context, req *apiv2beta1.CreateTaskRequest, opts ...grpc.CallOption) (*apiv2beta1.PipelineTaskDetail, error) {
	return a.api.CreateTask(ctx, req)
}

func (a *TestRunServiceAdapter) UpdateTask(ctx context.Context, req *apiv2beta1.UpdateTaskRequest, opts ...grpc.CallOption) (*apiv2beta1.PipelineTaskDetail, error) {
	return a.api.UpdateTask(ctx, req)
}

func (a *TestRunServiceAdapter) GetTask(ctx context.Context, req *apiv2beta1.GetTaskRequest, opts ...grpc.CallOption) (*apiv2beta1.PipelineTaskDetail, error) {
	return a.api.GetTask(ctx, req)
}

func (a *TestRunServiceAdapter) ListTasks(ctx context.Context, req *apiv2beta1.ListTasksRequest, opts ...grpc.CallOption) (*apiv2beta1.ListTasksResponse, error) {
	return a.api.ListTasks(ctx, req)
}

// Unimplemented methods (not needed for launcher testing)
// These return errors if called, which helps us identify if launcher starts using them

func (a *TestRunServiceAdapter) CreateRun(ctx context.Context, req *apiv2beta1.CreateRunRequest, opts ...grpc.CallOption) (*apiv2beta1.Run, error) {
	return nil, unimplementedError("CreateRun")
}

func (a *TestRunServiceAdapter) ListRuns(ctx context.Context, req *apiv2beta1.ListRunsRequest, opts ...grpc.CallOption) (*apiv2beta1.ListRunsResponse, error) {
	return nil, unimplementedError("ListRuns")
}

func (a *TestRunServiceAdapter) ArchiveRun(ctx context.Context, req *apiv2beta1.ArchiveRunRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, unimplementedError("ArchiveRun")
}

func (a *TestRunServiceAdapter) UnarchiveRun(ctx context.Context, req *apiv2beta1.UnarchiveRunRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, unimplementedError("UnarchiveRun")
}

func (a *TestRunServiceAdapter) DeleteRun(ctx context.Context, req *apiv2beta1.DeleteRunRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, unimplementedError("DeleteRun")
}

func (a *TestRunServiceAdapter) ReadArtifact(ctx context.Context, req *apiv2beta1.ReadArtifactRequest, opts ...grpc.CallOption) (*apiv2beta1.ReadArtifactResponse, error) {
	return nil, unimplementedError("ReadArtifact")
}

func (a *TestRunServiceAdapter) TerminateRun(ctx context.Context, req *apiv2beta1.TerminateRunRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, unimplementedError("TerminateRun")
}

func (a *TestRunServiceAdapter) RetryRun(ctx context.Context, req *apiv2beta1.RetryRunRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, unimplementedError("RetryRun")
}

func unimplementedError(method string) error {
	return &unimplementedErr{
		message: "TestRunServiceAdapter: " + method + " is not implemented for testing. This method is not needed for launcher testing.",
	}
}

type unimplementedErr struct {
	message string
}

func (e *unimplementedErr) Error() string {
	return e.message
}
