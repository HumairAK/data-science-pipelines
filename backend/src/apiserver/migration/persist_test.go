// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package migration

import (
	"context"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	mlmd "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPersistIsIdempotentAndValidatesRelationships(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:persist_test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(model.AllModels()...))

	runID := "run-1"
	taskID := "task-1"
	artifactID := "artifact-1"
	snapshot := &ConvertedSnapshot{
		Runs: []*model.Run{{
			UUID: runID, DisplayName: "run-1", K8SName: "run-1", Namespace: "team-a",
			StorageState: model.StorageStateAvailable,
		}},
		Tasks: []*model.Task{{
			UUID: taskID, Namespace: "team-a", RunUUID: runID, Name: "task-1",
			DisplayName: "task-1", State: model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED), Type: model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
			Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{},
		}},
		Artifacts: []*model.Artifact{{
			UUID: artifactID, Namespace: "team-a", URI: persistStringPtr("minio://mlpipeline/v2/artifacts/run-1/output"),
			Name: "output", IdentityKey: persistStringPtr("identity-1"),
		}},
		Relationships: []*model.ArtifactTask{{
			UUID: "relationship-1", ArtifactID: artifactID, TaskID: taskID,
			RunUUID: runID, Type: model.IOType(apiv2beta1.IOType_OUTPUT), Iteration: model.ArtifactTaskNoIteration,
		}},
	}

	first, err := Persist(db, snapshot)
	require.NoError(t, err)
	require.Equal(t, Counts{Runs: 1, Tasks: 1, Artifacts: 1, Relationships: 1, InsertedRuns: 1, InsertedTasks: 1, InsertedArtifacts: 1, InsertedRelationships: 1}, first)
	second, err := Persist(db, snapshot)
	require.NoError(t, err)
	require.Equal(t, Counts{Runs: 1, Tasks: 1, Artifacts: 1, Relationships: 1, ExistingRuns: 1, ExistingTasks: 1, ExistingArtifacts: 1, ExistingRelationships: 1}, second)

	var runs, tasks, artifacts, relationships int64
	require.NoError(t, db.Model(&model.Run{}).Count(&runs).Error)
	require.NoError(t, db.Model(&model.Task{}).Count(&tasks).Error)
	require.NoError(t, db.Model(&model.Artifact{}).Count(&artifacts).Error)
	require.NoError(t, db.Model(&model.ArtifactTask{}).Count(&relationships).Error)
	require.EqualValues(t, 1, runs)
	require.EqualValues(t, 1, tasks)
	require.EqualValues(t, 1, artifacts)
	require.EqualValues(t, 1, relationships)
}

func TestRunCompletesAndCanResumeWithoutDuplicates(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:run_migration_test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(model.AllModels()...))

	runContextID, executionID, artifactID := int64(1), int64(10), int64(20)
	runType, executionType := pipelineRunContextTypeName, containerExecutionTypeName
	state := mlmd.Execution_COMPLETE
	uri := "minio://mlpipeline/v2/artifacts/legacy/output"
	source := &fakeSource{
		contextPages:   [][]*mlmd.Context{{{Id: &runContextID, Type: &runType, CustomProperties: map[string]*mlmd.Value{keyNamespace: stringValue("team-a"), "run_uuid": stringValue("run-1")}}}},
		executionPages: [][]*mlmd.Execution{{{Id: &executionID, Type: &executionType, LastKnownState: &state, CustomProperties: map[string]*mlmd.Value{keyNamespace: stringValue("team-a"), keyTaskName: stringValue("task")}}}},
		artifactPages:  [][]*mlmd.Artifact{{{Id: &artifactID, Uri: &uri}}},
		contexts:       map[int64][]*mlmd.Context{executionID: {{Id: &runContextID, Type: &runType, CustomProperties: map[string]*mlmd.Value{keyNamespace: stringValue("team-a"), "run_uuid": stringValue("run-1")}}}},
		events:         []*mlmd.Event{{ExecutionId: &executionID, ArtifactId: &artifactID, Type: eventTypePtr(mlmd.Event_OUTPUT)}},
	}

	first, err := Run(context.Background(), db, source, 100, "test", false, nil)
	require.NoError(t, err)
	require.Equal(t, Counts{Runs: 1, Tasks: 1, Artifacts: 1, Relationships: 1, InsertedRuns: 1, InsertedTasks: 1, InsertedArtifacts: 1, InsertedRelationships: 1}, first)
	var marker model.RuntimeMetadataMigration
	require.NoError(t, db.First(&marker).Error)
	require.Equal(t, model.RuntimeMetadataMigrationCompleted, marker.Status)

	second, err := Run(context.Background(), db, source, 100, "test", false, nil)
	require.NoError(t, err)
	require.Equal(t, first, second)
	var taskCount, artifactCount, relationshipCount int64
	require.NoError(t, db.Model(&model.Task{}).Count(&taskCount).Error)
	require.NoError(t, db.Model(&model.Artifact{}).Count(&artifactCount).Error)
	require.NoError(t, db.Model(&model.ArtifactTask{}).Count(&relationshipCount).Error)
	require.EqualValues(t, 1, taskCount)
	require.EqualValues(t, 1, artifactCount)
	require.EqualValues(t, 1, relationshipCount)
}

func TestPersistRejectsCrossNamespaceRelationship(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:namespace_validation_test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(model.AllModels()...))
	runID, taskID, artifactID := "run-1", "task-1", "artifact-1"
	_, err = Persist(db, &ConvertedSnapshot{
		Runs:          []*model.Run{{UUID: runID, DisplayName: runID, K8SName: runID, Namespace: "team-a", StorageState: model.StorageStateAvailable}},
		Tasks:         []*model.Task{{UUID: taskID, Namespace: "team-a", RunUUID: runID, Name: "task", DisplayName: "task", Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{}, State: model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED), Type: model.TaskType(apiv2beta1.PipelineTask_RUNTIME)}},
		Artifacts:     []*model.Artifact{{UUID: artifactID, Namespace: "team-b", URI: persistStringPtr("s3://bucket/a"), IdentityKey: persistStringPtr("identity")}},
		Relationships: []*model.ArtifactTask{{UUID: "link", ArtifactID: artifactID, TaskID: taskID, RunUUID: runID, Type: model.IOType(apiv2beta1.IOType_OUTPUT), Iteration: model.ArtifactTaskNoIteration}},
	})
	require.Error(t, err)
}

func persistStringPtr(value string) *string { return &value }
