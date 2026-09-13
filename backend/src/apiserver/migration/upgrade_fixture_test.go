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

// TestProductionLikeUpgradeFixture exercises the complete operator path with
// the shapes called out by the MLMD-removal upgrade contract. It intentionally
// uses the real snapshot loader, converter, persistence transaction, and
// migration ledger; only the MLMD transport is replaced by a deterministic
// fixture source and the native database is isolated SQLite.
func TestProductionLikeUpgradeFixture(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:production_like_upgrade_fixture?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(model.AllModels()...))

	source := productionLikeFixtureSource()
	first, err := Run(context.Background(), db, source, 100, "fixture/1", false, nil)
	require.NoError(t, err)
	require.Equal(t, Counts{Runs: 2, Tasks: 22, Artifacts: 2, Relationships: 18, Metrics: 2, InsertedRuns: 2, InsertedTasks: 22, InsertedArtifacts: 2, InsertedRelationships: 18, InsertedMetrics: 2}, first)

	var marker model.RuntimeMetadataMigration
	require.NoError(t, db.First(&marker).Error)
	require.Equal(t, model.RuntimeMetadataMigrationCompleted, marker.Status)

	var taskCount int64
	require.NoError(t, db.Model(&model.Task{}).Count(&taskCount).Error)
	require.EqualValues(t, 22, taskCount)

	var metricCount int64
	require.NoError(t, db.Model(&model.RunMetricV1{}).Count(&metricCount).Error)
	require.EqualValues(t, 2, metricCount)

	// Only completed and cached executions retain a cache fingerprint. Failed,
	// running, and cancelled executions are deliberately cache-ineligible.
	var eligible, ineligible int64
	require.NoError(t, db.Model(&model.Task{}).Where("Fingerprint <> ''").Count(&eligible).Error)
	require.NoError(t, db.Model(&model.Task{}).Where("Fingerprint = ''").Count(&ineligible).Error)
	require.EqualValues(t, 4, eligible)
	require.EqualValues(t, 18, ineligible)

	// Exercise every relationship semantic that the converter derives from
	// historical execution metadata, across both namespaces.
	for _, relationshipType := range []struct {
		name string
		io   apiv2beta1.IOType
		want int64
	}{
		{name: "iterator output", io: apiv2beta1.IOType_ITERATOR_OUTPUT, want: 2},
		{name: "iterator input", io: apiv2beta1.IOType_ITERATOR_INPUT, want: 2},
		{name: "one of output", io: apiv2beta1.IOType_ONE_OF_OUTPUT, want: 2},
		{name: "collected input", io: apiv2beta1.IOType_COLLECTED_INPUTS, want: 2},
		{name: "task final status", io: apiv2beta1.IOType_TASK_FINAL_STATUS_OUTPUT, want: 2},
	} {
		t.Run(relationshipType.name, func(t *testing.T) {
			var count int64
			require.NoError(t, db.Model(&model.ArtifactTask{}).
				Where("Type = ?", relationshipType.io).
				Count(&count).Error)
			require.Equal(t, relationshipType.want, count)
		})
	}

	var exitHandlerCount int64
	require.NoError(t, db.Model(&model.Task{}).
		Where("Name = ? AND Type = ?", "exit-handler-cleanup", apiv2beta1.PipelineTask_RUNTIME).
		Count(&exitHandlerCount).Error)
	require.EqualValues(t, 2, exitHandlerCount)

	// Every relationship must remain in the same tenant and run as both of its
	// endpoints. This also validates the historical unnamespaced URI path: the
	// URI itself contains no tenant, but the native row does.
	var crossNamespace int64
	require.NoError(t, db.Table("artifact_tasks AS links").
		Joins("JOIN tasks ON tasks.UUID = links.TaskID").
		Joins("JOIN artifacts ON artifacts.UUID = links.ArtifactID").
		Where("tasks.Namespace <> artifacts.Namespace OR tasks.RunUUID <> links.RunUUID").
		Count(&crossNamespace).Error)
	require.Zero(t, crossNamespace)

	var historical model.Artifact
	require.NoError(t, db.Where("URI = ? AND Namespace = ?", "minio://mlpipeline/v2/artifacts/shared/output", "team-a").First(&historical).Error)
	require.Equal(t, "team-a", historical.Namespace)

	second, err := Run(context.Background(), db, source, 100, "fixture/1", false, nil)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.NoError(t, db.Model(&model.Task{}).Count(&taskCount).Error)
	require.EqualValues(t, 22, taskCount)
}

func productionLikeFixtureSource() *fakeSource {
	contexts := make([]*mlmd.Context, 0, 2)
	contextByExecution := make(map[int64][]*mlmd.Context)
	executions := make([]*mlmd.Execution, 0, 12)
	artifacts := make([]*mlmd.Artifact, 0, 4)
	events := make([]*mlmd.Event, 0, 12)

	for namespaceIndex, namespace := range []string{"team-a", "team-b"} {
		runID := int64(namespaceIndex + 1)
		runType := pipelineRunContextTypeName
		runUUID := "run-" + namespace
		runContext := &mlmd.Context{Id: &runID, Type: &runType, Name: stringPtr(runUUID), CustomProperties: map[string]*mlmd.Value{
			keyNamespace: stringValue(namespace), "run_uuid": stringValue(runUUID), keyDisplayName: stringValue("production-like-" + namespace),
		}}
		contexts = append(contexts, runContext)

		base := int64((namespaceIndex + 1) * 100)
		rootID, loopID, conditionID, branchID, exitID := base+1, base+2, base+3, base+4, base+5
		failedID, runningID, cachedID, dagChildID, metricTaskID, canceledID := base+6, base+7, base+8, base+9, base+10, base+11
		complete := mlmd.Execution_COMPLETE
		failed := mlmd.Execution_FAILED
		active := mlmd.Execution_RUNNING
		cached := mlmd.Execution_CACHED
		canceled := mlmd.Execution_CANCELED
		types := map[int64]string{
			rootID: dagExecutionTypeName, loopID: "system.LoopExecution", conditionID: "system.ConditionExecution",
			branchID: "system.ConditionBranchExecution", exitID: containerExecutionTypeName, failedID: containerExecutionTypeName,
			runningID: containerExecutionTypeName, cachedID: containerExecutionTypeName, dagChildID: containerExecutionTypeName,
			metricTaskID: containerExecutionTypeName, canceledID: containerExecutionTypeName,
		}
		states := map[int64]*mlmd.Execution_State{
			rootID: &complete, loopID: &complete, conditionID: &complete, branchID: &complete, exitID: &complete,
			failedID: &failed, runningID: &active, cachedID: &cached, dagChildID: &complete, metricTaskID: &complete, canceledID: &canceled,
		}
		names := map[int64]string{
			rootID: "root-dag", loopID: "loop-body", conditionID: "condition", branchID: "condition-branch",
			exitID: "exit-handler-cleanup", failedID: "failed-task", runningID: "running-task", cachedID: "cached-task",
			dagChildID: "dag-child", metricTaskID: "metric-task", canceledID: "canceled-task",
		}
		parents := map[int64]int64{loopID: rootID, conditionID: rootID, branchID: conditionID, exitID: rootID, dagChildID: rootID}
		for executionID, executionType := range types {
			properties := map[string]*mlmd.Value{keyNamespace: stringValue(namespace), keyTaskName: stringValue(names[executionID])}
			if parentID := parents[executionID]; parentID != 0 {
				properties[keyParentDAGID] = intValue(parentID)
			}
			if executionID == loopID {
				properties[keyTaskType] = stringValue("loop")
				properties[keyIteration] = intValue(2)
				properties[keyIterator] = fixtureBoolValue(true)
			}
			if executionID == branchID {
				properties[keyTaskType] = stringValue("condition_branch")
				properties[keyOneOf] = fixtureBoolValue(true)
			}
			if executionID == dagChildID {
				properties[keyCollected] = fixtureBoolValue(true)
			}
			if executionID == runningID {
				properties[keyIterator] = fixtureBoolValue(true)
			}
			if executionID == exitID {
				properties[keyFinalStatus] = fixtureBoolValue(true)
			}
			if executionID == cachedID || executionID == dagChildID {
				properties[keyCacheFingerprint] = stringValue("cache-" + namespace)
			}
			execution := &mlmd.Execution{Id: &executionID, Type: stringPtr(executionType), LastKnownState: states[executionID], CustomProperties: properties}
			executions = append(executions, execution)
			contextByExecution[executionID] = []*mlmd.Context{runContext}
		}

		outputID := int64(namespaceIndex*2 + 1)
		metricID := int64(namespaceIndex*2 + 2)
		outputURI := "minio://mlpipeline/v2/artifacts/shared/output"
		metricURI := "minio://mlpipeline/v2/artifacts/" + namespace + "/metrics"
		artifacts = append(artifacts,
			&mlmd.Artifact{Id: &outputID, Uri: &outputURI, Type: stringPtr("system.Model")},
			&mlmd.Artifact{Id: &metricID, Uri: &metricURI, Type: stringPtr("system.SlicedClassificationMetric"), CustomProperties: map[string]*mlmd.Value{"double_value": doubleValue(0.9 + float64(namespaceIndex)/100)}},
		)
		for _, executionID := range []int64{rootID, loopID, conditionID, branchID, exitID, failedID} {
			events = append(events, &mlmd.Event{ExecutionId: &executionID, ArtifactId: &outputID, Type: eventTypePtr(mlmd.Event_OUTPUT)})
		}
		for _, executionID := range []int64{dagChildID, metricTaskID, runningID} {
			events = append(events, &mlmd.Event{ExecutionId: &executionID, ArtifactId: &outputID, Type: eventTypePtr(mlmd.Event_INPUT)})
		}
		events = append(events, &mlmd.Event{ExecutionId: &metricTaskID, ArtifactId: &metricID, Type: eventTypePtr(mlmd.Event_OUTPUT)})
	}
	return &fakeSource{
		contextPages:   [][]*mlmd.Context{contexts},
		executionPages: [][]*mlmd.Execution{executions},
		artifactPages:  [][]*mlmd.Artifact{artifacts},
		contexts:       contextByExecution,
		events:         events,
	}
}

func fixtureBoolValue(value bool) *mlmd.Value {
	return &mlmd.Value{Value: &mlmd.Value_BoolValue{BoolValue: value}}
}
