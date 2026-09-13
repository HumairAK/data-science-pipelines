// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package migration

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	mlmd "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	pipelineContextTypeName    = "system.Pipeline"
	pipelineRunContextTypeName = "system.PipelineRun"
	containerExecutionTypeName = "system.ContainerExecution"
	dagExecutionTypeName       = "system.DAGExecution"
	importerExecutionTypeName  = "system.ImporterExecution"

	keyNamespace        = "namespace"
	keyTaskName         = "task_name"
	keyDisplayName      = "display_name"
	keyParentDAGID      = "parent_dag_id"
	keyParentTaskID     = "parent_task_id"
	keyCacheFingerprint = "cache_fingerprint"
	keyInputs           = "inputs"
	keyOutputs          = "outputs"
	keyTaskType         = "task_type"
	keyIteration        = "iteration"
	keyScopePath        = "scope_path"
	keyIOType           = "io_type"
	keyFinalStatus      = "final_status"
	keyOneOf            = "one_of"
	keyCollected        = "collected"
	keyIterator         = "iterator"
)

// ConvertedSnapshot is the storage-neutral result of translating MLMD. The
// persistence phase can safely retry these records because all IDs and logical
// keys are deterministic source mappings.
type ConvertedSnapshot struct {
	Runs          []*model.Run
	Tasks         []*model.Task
	Artifacts     []*model.Artifact
	Relationships []*model.ArtifactTask
	Metrics       []*model.RunMetricV1
	Skipped       []MigrationIssue
	Unsupported   []MigrationIssue
}

// MigrationIssue makes omissions visible to operators instead of silently
// claiming that every MLMD record was migrated.
type MigrationIssue struct {
	Kind     string `json:"kind"`
	SourceID int64  `json:"source_id"`
	Reason   string `json:"reason"`
}

// ConvertSnapshot translates runtime MLMD entities into native KFP models.
// It deliberately keeps artifact URIs byte-for-byte unchanged, including old
// unnamespaced minio://mlpipeline/v2/artifacts/... URIs.
func ConvertSnapshot(snapshot *Snapshot) (*ConvertedSnapshot, error) {
	if snapshot == nil {
		return nil, fmt.Errorf("cannot convert a nil MLMD snapshot")
	}
	result := &ConvertedSnapshot{}
	runsByUUID := make(map[string]*model.Run)
	for _, contexts := range snapshot.ContextsByExecID {
		for _, context := range contexts {
			if context == nil || context.GetType() != pipelineRunContextTypeName {
				continue
			}
			runID := runUUID([]*mlmd.Context{context})
			if _, ok := runsByUUID[runID]; ok {
				continue
			}
			namespace := contextNamespace([]*mlmd.Context{context})
			displayName := valueString(context.GetCustomProperties(), keyDisplayName)
			if displayName == "" {
				displayName = context.GetName()
			}
			if displayName == "" {
				displayName = runID
			}
			pipelineManifest := contextString(contexts, "pipeline_spec_manifest", "pipeline_spec")
			workflowManifest := contextString(contexts, "workflow_spec_manifest", "workflow_spec")
			run := &model.Run{
				UUID:           runID,
				DisplayName:    displayName,
				K8SName:        runID,
				Description:    valueString(context.GetCustomProperties(), "description"),
				Namespace:      namespace,
				ServiceAccount: contextString(contexts, "service_account", "serviceAccount"),
				PipelineSpec: model.PipelineSpec{
					PipelineId:           contextString(contexts, "pipeline_id", "pipeline_uuid"),
					PipelineVersionId:    contextString(contexts, "pipeline_version_id", "pipeline_version_uuid"),
					PipelineName:         contextString(contexts, "pipeline_name"),
					PipelineSpecManifest: model.LargeText(pipelineManifest),
					WorkflowSpecManifest: model.LargeText(workflowManifest),
					Parameters:           model.LargeText(contextString(contexts, "parameters", "runtime_parameters")),
				},
				ExperimentId:            contextString(contexts, "experiment_id", "experiment_uuid"),
				StorageState:            model.StorageStateAvailable,
				PipelineRuntimeManifest: model.LargeText(""),
				WorkflowRuntimeManifest: model.LargeText(""),
			}
			runsByUUID[runID] = run
			result.Runs = append(result.Runs, run)
		}
	}
	tasksByExecution := make(map[int64]*model.Task)
	for _, execution := range snapshot.Executions {
		if execution == nil {
			result.Skipped = append(result.Skipped, MigrationIssue{Kind: "execution", Reason: "nil execution"})
			continue
		}
		if execution.GetId() == 0 {
			result.Skipped = append(result.Skipped, MigrationIssue{Kind: "execution", Reason: "missing MLMD execution id"})
			continue
		}
		if !isRuntimeExecution(execution.GetType()) {
			result.Unsupported = append(result.Unsupported, MigrationIssue{Kind: "execution", SourceID: execution.GetId(), Reason: fmt.Sprintf("unsupported execution type %q", execution.GetType())})
			continue
		}
		runID := runUUID(snapshot.ContextsByExecID[execution.GetId()])
		namespace := valueString(execution.GetCustomProperties(), keyNamespace)
		if namespace == "" {
			namespace = contextNamespace(snapshot.ContextsByExecID[execution.GetId()])
		}
		if namespace == "" {
			return nil, fmt.Errorf("cannot establish namespace for MLMD execution %d", execution.GetId())
		}
		name := valueString(execution.GetCustomProperties(), keyTaskName)
		if name == "" {
			name = execution.GetName()
		}
		if name == "" {
			name = fmt.Sprintf("mlmd-execution-%d", execution.GetId())
		}
		logicalKey := fmt.Sprintf("mlmd:%d:%s:%s", execution.GetId(), namespace, runID)
		created := millisToSeconds(execution.GetCreateTimeSinceEpoch())
		task := &model.Task{
			UUID:           util.NewDeterministicUUID(logicalKey),
			Namespace:      namespace,
			RunUUID:        runID,
			CreatedAtInSec: created,
			StartedInSec:   created,
			FinishedInSec:  millisToSeconds(execution.GetLastUpdateTimeSinceEpoch()),
			Fingerprint:    completedFingerprint(execution),
			Name:           name,
			DisplayName:    valueString(execution.GetCustomProperties(), keyDisplayName),
			State:          model.TaskStatus(taskState(execution.GetLastKnownState())),
			Type:           model.TaskType(taskType(execution.GetType(), execution.GetCustomProperties(), name)),
			Pods:           model.JSONSlice{},
			TypeAttrs:      taskAttributes(execution),
			ScopePath:      valueString(execution.GetCustomProperties(), keyScopePath),
			LogicalKey:     &logicalKey,
		}
		stateHistory, historyErr := taskStateHistory(task)
		if historyErr != nil {
			return nil, fmt.Errorf("convert state history for MLMD execution %d: %w", execution.GetId(), historyErr)
		}
		task.StateHistory = stateHistory
		if _, ok := runsByUUID[runID]; !ok {
			// Some older MLMD stores do not return pipeline-run contexts for
			// every execution. Keep the task referentially valid while retaining
			// the deterministic run identity for later reconciliation.
			runsByUUID[runID] = &model.Run{
				UUID: runID, DisplayName: runID, K8SName: runID,
				Namespace: namespace, StorageState: model.StorageStateAvailable,
				PipelineRuntimeManifest: model.LargeText(""),
				WorkflowRuntimeManifest: model.LargeText(""),
			}
			result.Runs = append(result.Runs, runsByUUID[runID])
		}
		if task.DisplayName == "" {
			task.DisplayName = task.Name
		}
		parentID := valueInt(execution.GetCustomProperties(), keyParentDAGID)
		if parentID == 0 {
			parentID = valueInt(execution.GetCustomProperties(), keyParentTaskID)
		}
		if parentID != 0 {
			parentKey := fmt.Sprintf("mlmd:%d:%s:%s", parentID, namespace, runID)
			parentUUID := util.NewDeterministicUUID(parentKey)
			task.ParentTaskUUID = &parentUUID
		}
		task.InputParameters = jsonSlice(execution.GetCustomProperties(), keyInputs)
		task.OutputParameters = jsonSlice(execution.GetCustomProperties(), keyOutputs)
		result.Tasks = append(result.Tasks, task)
		tasksByExecution[execution.GetId()] = task
	}
	if err := finalizeRunMetadata(runsByUUID, result.Tasks); err != nil {
		return nil, err
	}

	artifactsByID := make(map[int64]*model.Artifact)
	for _, artifact := range snapshot.Artifacts {
		if artifact == nil {
			result.Skipped = append(result.Skipped, MigrationIssue{Kind: "artifact", Reason: "nil artifact"})
			continue
		}
		if artifact.GetId() == 0 {
			result.Skipped = append(result.Skipped, MigrationIssue{Kind: "artifact", Reason: "missing MLMD artifact id"})
			continue
		}
		namespace := artifactNamespace(artifact, snapshot, tasksByExecution)
		if namespace == "" {
			return nil, fmt.Errorf("cannot establish namespace for MLMD artifact %d", artifact.GetId())
		}
		uri := artifact.GetUri()
		identity := fmt.Sprintf("mlmd-artifact:%d:%s", artifact.GetId(), namespace)
		native := &model.Artifact{
			UUID:            util.NewDeterministicUUID(identity),
			Namespace:       namespace,
			URI:             &uri,
			Name:            artifact.GetName(),
			CreatedAtInSec:  millisToSeconds(artifact.GetCreateTimeSinceEpoch()),
			LastUpdateInSec: millisToSeconds(artifact.GetCreateTimeSinceEpoch()),
			Metadata:        valuesToJSON(artifact.GetCustomProperties()),
			IdentityKey:     &identity,
		}
		if artifactType, numberValue, ok := metricArtifact(artifact); ok {
			native.Type = artifactType
			native.NumberValue = &numberValue
		}
		artifactsByID[artifact.GetId()] = native
		// Metric artifacts are represented by the native run-metrics API. Keep
		// them in the in-memory map so their producer can be located below, but
		// never persist them as ordinary downloadable artifacts.
		if _, _, isMetric := metricArtifact(artifact); !isMetric {
			result.Artifacts = append(result.Artifacts, native)
		}
	}
	producers := make(map[int64]model.JSONData)
	producerTasks := make(map[int64]*model.Task)
	for executionID, events := range snapshot.EventsByExecution {
		task := tasksByExecution[executionID]
		if task == nil {
			continue
		}
		for _, event := range events {
			if event != nil && isOutputEvent(event.GetType()) {
				producers[event.GetArtifactId()] = producerForTask(task)
				producerTasks[event.GetArtifactId()] = task
			}
		}
	}
	for executionID, events := range snapshot.EventsByExecution {
		task := tasksByExecution[executionID]
		if task == nil {
			continue
		}
		for eventIndex, event := range events {
			if event == nil {
				continue
			}
			artifact := artifactsByID[event.GetArtifactId()]
			if artifact == nil {
				result.Skipped = append(result.Skipped, MigrationIssue{Kind: "event", SourceID: event.GetArtifactId(), Reason: "event references an unmigrated artifact"})
				continue
			}
			if artifact.NumberValue != nil {
				continue
			}
			ioType := relationshipIOType(event, task)
			producer := producers[event.GetArtifactId()]
			if isOutputEvent(event.GetType()) {
				producer = producerForTask(task)
			}
			path := eventPath(event)
			iteration := taskIteration(task)
			result.Relationships = append(result.Relationships, &model.ArtifactTask{
				UUID:        util.NewDeterministicUUID(fmt.Sprintf("mlmd-event:%d:%d:%d:%s:%d", executionID, event.GetArtifactId(), event.GetType(), path, eventIndex)),
				ArtifactID:  artifact.UUID,
				TaskID:      task.UUID,
				Type:        model.IOType(ioType),
				Iteration:   iteration,
				RunUUID:     task.RunUUID,
				Producer:    producer,
				ArtifactKey: path,
			})
		}
	}
	for artifactID, artifact := range artifactsByID {
		if artifact.NumberValue == nil {
			continue
		}
		task := producerTasks[artifactID]
		if task == nil {
			continue
		}
		payload, err := json.Marshal(artifact.Metadata)
		if err != nil {
			return nil, fmt.Errorf("encode metric metadata for artifact %d: %w", artifactID, err)
		}
		name := artifact.Name
		if name == "" {
			name = fmt.Sprintf("mlmd-metric-%d", artifactID)
		}
		result.Metrics = append(result.Metrics, &model.RunMetricV1{
			RunUUID:     task.RunUUID,
			NodeID:      task.Name,
			Name:        name,
			NumberValue: *artifact.NumberValue,
			Format:      "RAW",
			Payload:     model.LargeText(payload),
		})
	}
	return result, nil
}

func contextString(contexts []*mlmd.Context, keys ...string) string {
	for _, context := range contexts {
		if context == nil {
			continue
		}
		for _, key := range keys {
			if value := valueString(context.GetCustomProperties(), key); value != "" {
				return value
			}
		}
	}
	return ""
}

// relationshipIOType maps the limited execution metadata retained by older
// MLMD writers to the native KFP relationship contract. Explicit io_type
// metadata wins; the remaining mappings are derived from task semantics.
func relationshipIOType(event *mlmd.Event, task *model.Task) apiv2beta1.IOType {
	if task != nil && task.TypeAttrs != nil {
		if raw, ok := task.TypeAttrs[keyIOType].(string); ok {
			if ioType, exists := apiv2beta1.IOType_value[strings.ToUpper(raw)]; exists {
				return apiv2beta1.IOType(ioType)
			}
		}
	}
	isOutput := isOutputEvent(event.GetType())
	if !isOutput {
		if taskAttributeBool(task, keyIterator) {
			return apiv2beta1.IOType_ITERATOR_INPUT
		}
		return apiv2beta1.IOType_COMPONENT_INPUT
	}
	if taskAttributeBool(task, keyFinalStatus) {
		return apiv2beta1.IOType_TASK_FINAL_STATUS_OUTPUT
	}
	if taskAttributeBool(task, keyOneOf) || task.Type == model.TaskType(apiv2beta1.PipelineTask_CONDITION_BRANCH) {
		return apiv2beta1.IOType_ONE_OF_OUTPUT
	}
	if taskAttributeBool(task, keyCollected) {
		return apiv2beta1.IOType_COLLECTED_INPUTS
	}
	if taskIteration(task) != model.ArtifactTaskNoIteration {
		return apiv2beta1.IOType_ITERATOR_OUTPUT
	}
	return apiv2beta1.IOType_OUTPUT
}

func taskAttributeBool(task *model.Task, key string) bool {
	if task == nil || task.TypeAttrs == nil {
		return false
	}
	switch value := task.TypeAttrs[key].(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(value, "true") || value == "1"
	default:
		return false
	}
}

// taskStateHistory preserves the last state observed by MLMD. MLMD exposes a
// last-known state rather than KFP's complete transition log, so this records
// an honest one-entry history instead of manufacturing intermediate states.
func taskStateHistory(task *model.Task) (model.JSONSlice, error) {
	update := task.FinishedInSec
	if update == 0 {
		update = task.StartedInSec
	}
	return model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTask_TaskStatus{{
		UpdateTime: timestamppb.New(time.Unix(update, 0)),
		State:      apiv2beta1.PipelineTask_TaskState(task.State),
	}})
}

// finalizeRunMetadata derives run-level lifecycle fields from the migrated
// tasks. This keeps native run queries and historical graphs useful even when
// the source store has no dedicated pipeline-run execution entity.
func finalizeRunMetadata(runs map[string]*model.Run, tasks []*model.Task) error {
	tasksByRun := make(map[string][]*model.Task)
	for _, task := range tasks {
		if task != nil {
			tasksByRun[task.RunUUID] = append(tasksByRun[task.RunUUID], task)
		}
	}
	for runID, run := range runs {
		runTasks := tasksByRun[runID]
		if len(runTasks) == 0 {
			continue
		}
		state := aggregateRunState(runTasks)
		run.State = state
		run.Conditions = string(state.ToV1())
		var latest int64
		for _, task := range runTasks {
			if run.CreatedAtInSec == 0 || (task.CreatedAtInSec > 0 && task.CreatedAtInSec < run.CreatedAtInSec) {
				run.CreatedAtInSec = task.CreatedAtInSec
			}
			if task.FinishedInSec > latest {
				latest = task.FinishedInSec
			}
		}
		if isTerminalRunState(state) {
			run.FinishedAtInSec = latest
		}
		run.StateHistory = []*model.RuntimeStatus{{UpdateTimeInSec: latest, State: state}}
		history, err := json.Marshal(run.StateHistory)
		if err != nil {
			return fmt.Errorf("encode state history for run %q: %w", runID, err)
		}
		run.StateHistoryString = model.LargeText(history)
	}
	return nil
}

func aggregateRunState(tasks []*model.Task) model.RuntimeState {
	allSkipped := true
	for _, task := range tasks {
		switch apiv2beta1.PipelineTask_TaskState(task.State) {
		case apiv2beta1.PipelineTask_FAILED:
			return model.RuntimeStateFailed
		case apiv2beta1.PipelineTask_RUNNING, apiv2beta1.PipelineTask_RUNTIME_STATE_UNSPECIFIED:
			return model.RuntimeStateRunning
		case apiv2beta1.PipelineTask_SUCCEEDED:
			allSkipped = false
		}
	}
	if allSkipped {
		return model.RuntimeStateSkipped
	}
	return model.RuntimeStateSucceeded
}

func isTerminalRunState(state model.RuntimeState) bool {
	return state == model.RuntimeStateSucceeded || state == model.RuntimeStateSkipped || state == model.RuntimeStateFailed || state == model.RuntimeStateCanceled
}

func metricArtifact(artifact *mlmd.Artifact) (model.ArtifactType, float64, bool) {
	typeName := strings.ToLower(artifact.GetType())
	if !strings.Contains(typeName, "metric") &&
		artifact.GetCustomProperties()["metric_value"] == nil &&
		artifact.GetCustomProperties()["double_value"] == nil {
		return 0, 0, false
	}
	for _, key := range []string{"metric_value", "double_value", "value"} {
		if value := artifact.GetCustomProperties()[key]; value != nil {
			return metricType(typeName), value.GetDoubleValue(), true
		}
	}
	return metricType(typeName), 0, true
}

func metricType(typeName string) model.ArtifactType {
	switch {
	case strings.Contains(typeName, "sliced"):
		return model.ArtifactType(apiv2beta1.Artifact_SlicedClassificationMetric)
	case strings.Contains(typeName, "classification"):
		return model.ArtifactType(apiv2beta1.Artifact_ClassificationMetric)
	default:
		return model.ArtifactType(apiv2beta1.Artifact_Metric)
	}
}

func taskAttributes(execution *mlmd.Execution) model.JSONData {
	attributes := valuesToJSON(execution.GetCustomProperties())
	attributes["mlmd_execution_id"] = execution.GetId()
	attributes["mlmd_type"] = execution.GetType()
	name := valueString(execution.GetCustomProperties(), keyTaskName)
	if strings.HasPrefix(strings.ToLower(name), "exit-handler-") || valueString(execution.GetCustomProperties(), "exit_handler") == "true" {
		attributes["is_exit_handler"] = true
	}
	return attributes
}

func isOutputEvent(eventType mlmd.Event_Type) bool {
	return eventType == mlmd.Event_OUTPUT || eventType == mlmd.Event_DECLARED_OUTPUT || eventType == mlmd.Event_INTERNAL_OUTPUT
}

func producerForTask(task *model.Task) model.JSONData {
	producer := model.JSONData{"taskName": task.Name}
	if iteration := taskIteration(task); iteration != model.ArtifactTaskNoIteration {
		producer["iteration"] = iteration
	}
	return producer
}

func taskIteration(task *model.Task) int64 {
	if task == nil || task.TypeAttrs == nil {
		return model.ArtifactTaskNoIteration
	}
	if value, ok := task.TypeAttrs[keyIteration].(float64); ok {
		return int64(value)
	}
	if value, ok := task.TypeAttrs[keyIteration].(int64); ok {
		return value
	}
	if value, ok := task.TypeAttrs[keyIteration].(int); ok {
		return int64(value)
	}
	return model.ArtifactTaskNoIteration
}

func isRuntimeExecution(executionType string) bool {
	return executionType == containerExecutionTypeName || executionType == dagExecutionTypeName || executionType == importerExecutionTypeName || strings.HasPrefix(executionType, "system.")
}

func taskType(executionType string, properties map[string]*mlmd.Value, taskName string) apiv2beta1.PipelineTask_TaskType {
	explicit := strings.ToLower(valueString(properties, keyTaskType))
	switch {
	case strings.Contains(explicit, "condition_branch"), strings.Contains(explicit, "condition-branch"):
		return apiv2beta1.PipelineTask_CONDITION_BRANCH
	case strings.Contains(explicit, "condition"):
		return apiv2beta1.PipelineTask_CONDITION
	case strings.Contains(explicit, "loop"), strings.Contains(explicit, "for"):
		return apiv2beta1.PipelineTask_LOOP
	case strings.Contains(explicit, "import"):
		return apiv2beta1.PipelineTask_IMPORTER
	case strings.Contains(explicit, "dag"):
		return apiv2beta1.PipelineTask_DAG
	}
	typeName := strings.ToLower(executionType)
	switch {
	case strings.Contains(typeName, "conditionbranch") || strings.Contains(typeName, "condition_branch"):
		return apiv2beta1.PipelineTask_CONDITION_BRANCH
	case strings.Contains(typeName, "condition"):
		return apiv2beta1.PipelineTask_CONDITION
	case strings.Contains(typeName, "loop") || strings.Contains(typeName, "for"):
		return apiv2beta1.PipelineTask_LOOP
	case executionType == dagExecutionTypeName:
		return apiv2beta1.PipelineTask_DAG
	case executionType == importerExecutionTypeName:
		return apiv2beta1.PipelineTask_IMPORTER
	case strings.HasPrefix(strings.ToLower(taskName), "exit-handler-"):
		return apiv2beta1.PipelineTask_RUNTIME
	default:
		return apiv2beta1.PipelineTask_RUNTIME
	}
}

func taskState(state mlmd.Execution_State) apiv2beta1.PipelineTask_TaskState {
	switch state {
	case mlmd.Execution_COMPLETE, mlmd.Execution_CACHED:
		return apiv2beta1.PipelineTask_SUCCEEDED
	case mlmd.Execution_FAILED:
		return apiv2beta1.PipelineTask_FAILED
	case mlmd.Execution_CANCELED:
		return apiv2beta1.PipelineTask_SKIPPED
	default:
		return apiv2beta1.PipelineTask_RUNNING
	}
}

func completedFingerprint(execution *mlmd.Execution) string {
	if execution.GetLastKnownState() != mlmd.Execution_COMPLETE && execution.GetLastKnownState() != mlmd.Execution_CACHED {
		return ""
	}
	return valueString(execution.GetCustomProperties(), keyCacheFingerprint)
}

func runUUID(contexts []*mlmd.Context) string {
	for _, context := range contexts {
		if context != nil && context.GetType() == pipelineRunContextTypeName {
			if id := valueString(context.GetCustomProperties(), "run_uuid"); id != "" {
				return id
			}
			return util.NewDeterministicUUID("mlmd-context:" + strconv.FormatInt(context.GetId(), 10))
		}
	}
	return util.NewDeterministicUUID("mlmd-orphan-run")
}

func contextNamespace(contexts []*mlmd.Context) string {
	for _, context := range contexts {
		if context != nil {
			if namespace := valueString(context.GetCustomProperties(), keyNamespace); namespace != "" {
				return namespace
			}
		}
	}
	return ""
}

func artifactNamespace(artifact *mlmd.Artifact, snapshot *Snapshot, tasks map[int64]*model.Task) string {
	// Artifact properties are not an ownership proof. Resolve ownership only
	// through event execution associations and trusted PipelineRun contexts.
	namespaces := make(map[string]struct{})
	for _, event := range snapshot.EventsByExecution {
		for _, candidate := range event {
			if candidate != nil && candidate.GetArtifactId() == artifact.GetId() {
				if task := tasks[candidate.GetExecutionId()]; task != nil {
					namespaces[task.Namespace] = struct{}{}
				}
			}
		}
	}
	if len(namespaces) == 1 {
		for namespace := range namespaces {
			return namespace
		}
	}
	return ""
}

func valueString(values map[string]*mlmd.Value, key string) string {
	if value := values[key]; value != nil {
		return value.GetStringValue()
	}
	return ""
}

func valueInt(values map[string]*mlmd.Value, key string) int64 {
	if value := values[key]; value != nil {
		return value.GetIntValue()
	}
	return 0
}

func jsonSlice(values map[string]*mlmd.Value, key string) model.JSONSlice {
	if value := values[key]; value != nil {
		if value.GetStructValue() != nil {
			return model.JSONSlice{value.GetStructValue().AsMap()}
		}
		if raw := value.GetStringValue(); raw != "" {
			var decoded any
			if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
				return model.JSONSlice{decoded}
			}
		}
	}
	return nil
}

func valuesToJSON(values map[string]*mlmd.Value) model.JSONData {
	result := model.JSONData{}
	for key, value := range values {
		if value == nil {
			continue
		}
		switch typed := value.Value.(type) {
		case *mlmd.Value_StructValue:
			if typed.StructValue != nil {
				result[key] = typed.StructValue.AsMap()
			}
		case *mlmd.Value_StringValue:
			result[key] = typed.StringValue
		case *mlmd.Value_IntValue:
			result[key] = typed.IntValue
		case *mlmd.Value_DoubleValue:
			result[key] = typed.DoubleValue
		case *mlmd.Value_BoolValue:
			result[key] = typed.BoolValue
		}
	}
	return result
}

func eventPath(event *mlmd.Event) string {
	if event.GetPath() == nil {
		return ""
	}
	return event.GetPath().String()
}

func millisToSeconds(milliseconds int64) int64 {
	if milliseconds <= 0 {
		return 0
	}
	return milliseconds / 1000
}
