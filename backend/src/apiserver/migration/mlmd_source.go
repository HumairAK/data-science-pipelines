// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package migration

import (
	"context"
	"fmt"

	mlmd "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
)

// MLMDSource is the subset of MLMD read operations required by the migration.
// Keeping this interface separate from the generated client makes the
// conversion and resume logic testable without a live metadata service.
type MLMDSource interface {
	ListContexts(context.Context, *mlmd.ListOperationOptions) ([]*mlmd.Context, string, error)
	ListExecutions(context.Context, *mlmd.ListOperationOptions) ([]*mlmd.Execution, string, error)
	ListArtifacts(context.Context, *mlmd.ListOperationOptions) ([]*mlmd.Artifact, string, error)
	GetContextsByExecution(context.Context, int64) ([]*mlmd.Context, error)
	GetEventsByExecutionIDs(context.Context, []int64) ([]*mlmd.Event, error)
}

// GRPCSource adapts the generated MLMD client to MLMDSource.
type GRPCSource struct {
	client mlmd.MetadataStoreServiceClient
}

func NewGRPCSource(client mlmd.MetadataStoreServiceClient) *GRPCSource {
	return &GRPCSource{client: client}
}

func (s *GRPCSource) ListContexts(ctx context.Context, options *mlmd.ListOperationOptions) ([]*mlmd.Context, string, error) {
	response, err := s.client.GetContexts(ctx, &mlmd.GetContextsRequest{Options: options})
	if err != nil {
		return nil, "", fmt.Errorf("list MLMD contexts: %w", err)
	}
	return response.GetContexts(), response.GetNextPageToken(), nil
}

func (s *GRPCSource) ListExecutions(ctx context.Context, options *mlmd.ListOperationOptions) ([]*mlmd.Execution, string, error) {
	response, err := s.client.GetExecutions(ctx, &mlmd.GetExecutionsRequest{Options: options})
	if err != nil {
		return nil, "", fmt.Errorf("list MLMD executions: %w", err)
	}
	return response.GetExecutions(), response.GetNextPageToken(), nil
}

func (s *GRPCSource) ListArtifacts(ctx context.Context, options *mlmd.ListOperationOptions) ([]*mlmd.Artifact, string, error) {
	response, err := s.client.GetArtifacts(ctx, &mlmd.GetArtifactsRequest{Options: options})
	if err != nil {
		return nil, "", fmt.Errorf("list MLMD artifacts: %w", err)
	}
	return response.GetArtifacts(), response.GetNextPageToken(), nil
}

func (s *GRPCSource) GetContextsByExecution(ctx context.Context, executionID int64) ([]*mlmd.Context, error) {
	response, err := s.client.GetContextsByExecution(ctx, &mlmd.GetContextsByExecutionRequest{ExecutionId: &executionID})
	if err != nil {
		return nil, fmt.Errorf("get MLMD contexts for execution %d: %w", executionID, err)
	}
	return response.GetContexts(), nil
}

func (s *GRPCSource) GetEventsByExecutionIDs(ctx context.Context, executionIDs []int64) ([]*mlmd.Event, error) {
	if len(executionIDs) == 0 {
		return nil, nil
	}
	response, err := s.client.GetEventsByExecutionIDs(ctx, &mlmd.GetEventsByExecutionIDsRequest{ExecutionIds: executionIDs})
	if err != nil {
		return nil, fmt.Errorf("get MLMD events: %w", err)
	}
	return response.GetEvents(), nil
}

// Snapshot is an in-memory source view used by the converter. Pagination bounds
// individual MLMD responses, but this implementation still materializes the
// complete source view; operators should quiesce MLMD writers during migration.
type Snapshot struct {
	Contexts          []*mlmd.Context
	Executions        []*mlmd.Execution
	Artifacts         []*mlmd.Artifact
	ContextsByExecID  map[int64][]*mlmd.Context
	EventsByExecution map[int64][]*mlmd.Event
}

// LoadSnapshot reads all MLMD resources using pagination and groups the
// execution associations/events needed for task and relationship conversion.
// It is restart-safe, but not a transactional MLMD snapshot.
func LoadSnapshot(ctx context.Context, source MLMDSource, pageSize int32) (*Snapshot, error) {
	if pageSize <= 0 {
		return nil, fmt.Errorf("MLMD snapshot page size must be positive")
	}
	result := &Snapshot{
		ContextsByExecID:  make(map[int64][]*mlmd.Context),
		EventsByExecution: make(map[int64][]*mlmd.Event),
	}

	var err error
	result.Contexts, err = listAll(ctx, pageSize, source.ListContexts)
	if err != nil {
		return nil, err
	}
	result.Executions, err = listAll(ctx, pageSize, source.ListExecutions)
	if err != nil {
		return nil, err
	}
	result.Artifacts, err = listAll(ctx, pageSize, source.ListArtifacts)
	if err != nil {
		return nil, err
	}

	executionIDs := make([]int64, 0, len(result.Executions))
	for _, execution := range result.Executions {
		if execution == nil || execution.GetId() == 0 {
			continue
		}
		executionIDs = append(executionIDs, execution.GetId())
		contexts, contextErr := source.GetContextsByExecution(ctx, execution.GetId())
		if contextErr != nil {
			return nil, contextErr
		}
		result.ContextsByExecID[execution.GetId()] = contexts
	}

	events, err := source.GetEventsByExecutionIDs(ctx, executionIDs)
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		if event != nil {
			result.EventsByExecution[event.GetExecutionId()] = append(result.EventsByExecution[event.GetExecutionId()], event)
		}
	}
	return result, nil
}

func listAll[T any](ctx context.Context, pageSize int32, list func(context.Context, *mlmd.ListOperationOptions) ([]T, string, error)) ([]T, error) {
	var result []T
	var pageToken string
	for {
		token := pageToken
		options := &mlmd.ListOperationOptions{MaxResultSize: &pageSize}
		if token != "" {
			options.NextPageToken = &token
		}
		page, nextToken, err := list(ctx, options)
		if err != nil {
			return nil, err
		}
		result = append(result, page...)
		if nextToken == "" {
			return result, nil
		}
		if nextToken == pageToken {
			return nil, fmt.Errorf("MLMD pagination repeated token %q", nextToken)
		}
		pageToken = nextToken
	}
}
