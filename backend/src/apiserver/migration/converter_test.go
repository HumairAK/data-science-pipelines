// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package migration

import (
	"encoding/json"
	"strings"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	mlmd "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/require"
)

func TestConvertSnapshotPreservesArtifactURIAndCompletedCacheFingerprint(t *testing.T) {
	completed := mlmd.Execution_COMPLETE
	failed := mlmd.Execution_FAILED
	completedID, failedID, artifactID := int64(10), int64(11), int64(20)
	completedAt, updatedAt := int64(1_700_000_000_000), int64(1_700_000_001_000)
	uri := "minio://mlpipeline/v2/artifacts/legacy/run/output"
	snapshot := &Snapshot{
		Executions: []*mlmd.Execution{
			{Id: &completedID, Type: stringPtr(containerExecutionTypeName), LastKnownState: &completed, CreateTimeSinceEpoch: &completedAt, LastUpdateTimeSinceEpoch: &updatedAt, CustomProperties: map[string]*mlmd.Value{
				keyNamespace:        stringValue("team-a"),
				keyTaskName:         stringValue("producer"),
				keyCacheFingerprint: stringValue("fingerprint-1"),
			}},
			{Id: &failedID, Type: stringPtr(containerExecutionTypeName), LastKnownState: &failed, CustomProperties: map[string]*mlmd.Value{keyNamespace: stringValue("team-a"), keyTaskName: stringValue("failed")}},
		},
		Artifacts: []*mlmd.Artifact{{Id: &artifactID, Uri: &uri}},
		EventsByExecution: map[int64][]*mlmd.Event{
			10: {{ArtifactId: &artifactID, ExecutionId: &completedID, Type: eventTypePtr(mlmd.Event_OUTPUT)}},
		},
	}

	converted, err := ConvertSnapshot(snapshot)
	require.NoError(t, err)
	require.Len(t, converted.Tasks, 2)
	require.Len(t, converted.Artifacts, 1)
	require.Len(t, converted.Relationships, 1)

	require.Equal(t, "fingerprint-1", converted.Tasks[0].Fingerprint)
	require.Equal(t, "", converted.Tasks[1].Fingerprint)
	require.Equal(t, model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED), converted.Tasks[0].State)
	require.Len(t, converted.Tasks[0].StateHistory, 1)
	require.Equal(t, uri, *converted.Artifacts[0].URI)
	require.Equal(t, model.ArtifactType(apiv2beta1.Artifact_Artifact), converted.Artifacts[0].Type)
	require.Len(t, converted.Artifacts[0].URIHash, 64)
	require.Equal(t, model.IOType(apiv2beta1.IOType_OUTPUT), converted.Relationships[0].Type)
	require.Equal(t, "team-a", converted.Artifacts[0].Namespace)
}

func TestConvertSnapshotPreservesTaskSemanticsMetricsAndProducerLinks(t *testing.T) {
	runContextID := int64(1)
	producerID, consumerID, artifactID, metricID := int64(10), int64(11), int64(20), int64(21)
	complete := mlmd.Execution_COMPLETE
	runContextType := pipelineRunContextTypeName
	containerType := containerExecutionTypeName
	loopType := "system.LoopExecution"
	uri := "s3://bucket/output"
	metricURI := "file:///tmp/metric"
	contexts := []*mlmd.Context{{Id: &runContextID, Type: &runContextType, Name: stringPtr("run-1"), CustomProperties: map[string]*mlmd.Value{
		keyNamespace: stringValue("team-a"), "run_uuid": stringValue("run-1"),
		"pipeline_id": stringValue("pipeline-1"), "pipeline_version_id": stringValue("version-1"),
		"pipeline_name": stringValue("pipeline"), "parameters": stringValue(`{"input":"value"}`),
	}}}
	producer := &mlmd.Execution{Id: &producerID, Type: &containerType, LastKnownState: &complete, CustomProperties: map[string]*mlmd.Value{
		keyNamespace: stringValue("team-a"), keyTaskName: stringValue("producer"),
	}}
	consumer := &mlmd.Execution{Id: &consumerID, Type: &loopType, LastKnownState: &complete, CustomProperties: map[string]*mlmd.Value{
		keyNamespace: stringValue("team-a"), keyTaskName: stringValue("exit-handler-loop"), keyTaskType: stringValue("loop"),
		keyIteration: intValue(3),
	}}
	converted, err := ConvertSnapshot(&Snapshot{
		Contexts:   contexts,
		Executions: []*mlmd.Execution{producer, consumer},
		Artifacts: []*mlmd.Artifact{
			{Id: &artifactID, Uri: &uri, Type: stringPtr("system.Model")},
			{Id: &metricID, Uri: &metricURI, Name: stringPtr("metric"), Type: stringPtr("system.SlicedClassificationMetric"), CustomProperties: map[string]*mlmd.Value{"double_value": doubleValue(0.91)}},
		},
		ContextsByExecID: map[int64][]*mlmd.Context{producerID: {contexts[0]}, consumerID: {contexts[0]}},
		EventsByExecution: map[int64][]*mlmd.Event{
			producerID: {
				{ExecutionId: &producerID, ArtifactId: &artifactID, Type: eventTypePtr(mlmd.Event_OUTPUT)},
				{ExecutionId: &producerID, ArtifactId: &metricID, Type: eventTypePtr(mlmd.Event_OUTPUT)},
			},
			consumerID: {{ExecutionId: &consumerID, ArtifactId: &artifactID, Type: eventTypePtr(mlmd.Event_INPUT)}},
		},
	})
	require.NoError(t, err)
	require.Len(t, converted.Runs, 1)
	require.Equal(t, model.RuntimeStateSucceeded, converted.Runs[0].State)
	require.Len(t, converted.Runs[0].StateHistory, 1)
	require.Equal(t, "pipeline-1", converted.Runs[0].PipelineId)
	require.Equal(t, "version-1", converted.Runs[0].PipelineVersionId)
	require.Equal(t, "pipeline", converted.Runs[0].PipelineName)
	require.Len(t, converted.Tasks, 2)
	require.Equal(t, model.TaskType(apiv2beta1.PipelineTask_LOOP), converted.Tasks[1].Type)
	require.Equal(t, int64(3), taskIteration(converted.Tasks[1]))
	require.Len(t, converted.Artifacts, 1)
	require.Len(t, converted.Metrics, 1)
	require.Equal(t, model.ArtifactType(apiv2beta1.Artifact_Model), converted.Artifacts[0].Type)
	require.Len(t, converted.Artifacts[0].URIHash, 64)
	require.InDelta(t, 0.91, converted.Metrics[0].NumberValue, 0.0001)
	require.Contains(t, string(converted.Metrics[0].Payload), `"RunUUID":"run-1"`)
	require.Contains(t, string(converted.Metrics[0].Payload), `"NodeID":"producer"`)
	require.Contains(t, string(converted.Metrics[0].Payload), `"Name":"metric"`)
	require.Len(t, converted.Relationships, 2)
	var input model.ArtifactTask
	for _, relationship := range converted.Relationships {
		if relationship.Type == model.IOType(apiv2beta1.IOType_COMPONENT_INPUT) {
			input = *relationship
		}
	}
	require.Equal(t, "producer", input.Producer["taskName"])
	require.Equal(t, int64(3), input.Iteration)
	encoded, err := json.Marshal(converted.Tasks[1].TypeAttrs)
	require.NoError(t, err)
	require.Contains(t, string(encoded), "mlmd_type")
}

func TestConvertSnapshotKeepsOrphanRunsDistinctAndLogicalKeysBounded(t *testing.T) {
	complete := mlmd.Execution_COMPLETE
	firstID, secondID := int64(101), int64(102)
	longNamespace := strings.Repeat("n", 63)
	converted, err := ConvertSnapshot(&Snapshot{Executions: []*mlmd.Execution{
		{Id: &firstID, Type: stringPtr(containerExecutionTypeName), LastKnownState: &complete, CustomProperties: map[string]*mlmd.Value{
			keyNamespace: stringValue(longNamespace), keyTaskName: stringValue("first"),
		}},
		{Id: &secondID, Type: stringPtr(containerExecutionTypeName), LastKnownState: &complete, CustomProperties: map[string]*mlmd.Value{
			keyNamespace: stringValue(longNamespace), keyTaskName: stringValue("second"),
		}},
	}})
	require.NoError(t, err)
	require.Len(t, converted.Runs, 2)
	require.NotEqual(t, converted.Tasks[0].RunUUID, converted.Tasks[1].RunUUID)
	for _, task := range converted.Tasks {
		require.NotNil(t, task.LogicalKey)
		require.LessOrEqual(t, len(*task.LogicalKey), 64)
	}
}

func stringPtr(value string) *string { return &value }

func stringValue(value string) *mlmd.Value {
	return &mlmd.Value{Value: &mlmd.Value_StringValue{StringValue: value}}
}

func intValue(value int64) *mlmd.Value {
	return &mlmd.Value{Value: &mlmd.Value_IntValue{IntValue: value}}
}

func doubleValue(value float64) *mlmd.Value {
	return &mlmd.Value{Value: &mlmd.Value_DoubleValue{DoubleValue: value}}
}

func eventTypePtr(value mlmd.Event_Type) *mlmd.Event_Type { return &value }
