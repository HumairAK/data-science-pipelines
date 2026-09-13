// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package migration

import (
	"context"
	"testing"

	mlmd "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/require"
)

type fakeSource struct {
	contextPages   [][]*mlmd.Context
	executionPages [][]*mlmd.Execution
	artifactPages  [][]*mlmd.Artifact
	contexts       map[int64][]*mlmd.Context
	events         []*mlmd.Event
}

func (f *fakeSource) ListContexts(_ context.Context, options *mlmd.ListOperationOptions) ([]*mlmd.Context, string, error) {
	return page(f.contextPages, options.GetNextPageToken())
}

func (f *fakeSource) ListExecutions(_ context.Context, options *mlmd.ListOperationOptions) ([]*mlmd.Execution, string, error) {
	return page(f.executionPages, options.GetNextPageToken())
}

func (f *fakeSource) ListArtifacts(_ context.Context, options *mlmd.ListOperationOptions) ([]*mlmd.Artifact, string, error) {
	return page(f.artifactPages, options.GetNextPageToken())
}

func (f *fakeSource) GetContextsByExecution(_ context.Context, id int64) ([]*mlmd.Context, error) {
	return f.contexts[id], nil
}

func (f *fakeSource) GetEventsByExecutionIDs(_ context.Context, ids []int64) ([]*mlmd.Event, error) {
	allowed := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		allowed[id] = struct{}{}
	}
	filtered := make([]*mlmd.Event, 0, len(f.events))
	for _, event := range f.events {
		if event != nil {
			if _, ok := allowed[event.GetExecutionId()]; ok {
				filtered = append(filtered, event)
			}
		}
	}
	return filtered, nil
}

func page[T any](pages [][]T, token string) ([]T, string, error) {
	index := 0
	if token != "" {
		index = int(token[0] - '0')
	}
	if index >= len(pages) {
		return nil, "", nil
	}
	next := ""
	if index+1 < len(pages) {
		next = string(rune('0' + index + 1))
	}
	return pages[index], next, nil
}

func TestLoadSnapshotPaginatesAndGroupsRuntimeData(t *testing.T) {
	source := &fakeSource{
		contextPages:   [][]*mlmd.Context{{{Id: int64Ptr(1)}}, {{Id: int64Ptr(2)}}},
		executionPages: [][]*mlmd.Execution{{{Id: int64Ptr(10)}}, {{Id: int64Ptr(11)}}},
		artifactPages:  [][]*mlmd.Artifact{{{Id: int64Ptr(20)}}, {{Id: int64Ptr(21)}}},
		contexts:       map[int64][]*mlmd.Context{10: {{Id: int64Ptr(1)}}, 11: {{Id: int64Ptr(2)}}},
		events:         []*mlmd.Event{{ExecutionId: int64Ptr(10)}},
	}

	snapshot, err := LoadSnapshot(context.Background(), source, 1)
	require.NoError(t, err)
	require.Len(t, snapshot.Contexts, 2)
	require.Len(t, snapshot.Executions, 2)
	require.Len(t, snapshot.Artifacts, 2)
	require.Len(t, snapshot.ContextsByExecID[10], 1)
	require.Len(t, snapshot.EventsByExecution[10], 1)
}

func int64Ptr(value int64) *int64 { return &value }
