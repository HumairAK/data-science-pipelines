// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ProgressFunc receives a human-readable progress update from Run.
type ProgressFunc func(string, ...any)

// Counts describes the records written or verified by a migration run.
type Counts struct {
	Runs                  int64 `json:"runs"`
	Tasks                 int64 `json:"tasks"`
	Artifacts             int64 `json:"artifacts"`
	Relationships         int64 `json:"relationships"`
	Metrics               int64 `json:"metrics"`
	InsertedRuns          int64 `json:"inserted_runs"`
	InsertedTasks         int64 `json:"inserted_tasks"`
	InsertedArtifacts     int64 `json:"inserted_artifacts"`
	InsertedRelationships int64 `json:"inserted_relationships"`
	InsertedMetrics       int64 `json:"inserted_metrics"`
	ExistingRuns          int64 `json:"existing_runs"`
	ExistingTasks         int64 `json:"existing_tasks"`
	ExistingArtifacts     int64 `json:"existing_artifacts"`
	ExistingRelationships int64 `json:"existing_relationships"`
	ExistingMetrics       int64 `json:"existing_metrics"`
	Skipped               int64 `json:"skipped"`
	Unsupported           int64 `json:"unsupported"`
}

// Persist writes a converted snapshot using deterministic source identities.
// Re-running it is safe: rows are inserted by stable UUID and existing rows are
// left intact. The transaction also makes an interrupted batch invisible.
func Persist(db *gorm.DB, snapshot *ConvertedSnapshot) (Counts, error) {
	if db == nil || snapshot == nil {
		return Counts{}, fmt.Errorf("migration persistence requires a database and snapshot")
	}
	var counts Counts
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, run := range snapshot.Runs {
			if run == nil || run.UUID == "" {
				continue
			}
			result := tx.Omit("Tasks", "Metrics", "ResourceReferences").Clauses(ignoreConflict()).Create(run)
			if result.Error != nil {
				return fmt.Errorf("persist run %q: %w", run.UUID, result.Error)
			}
			counts.Runs++
			if result.RowsAffected == 1 {
				counts.InsertedRuns++
			} else {
				counts.ExistingRuns++
			}
		}
		for _, task := range snapshot.Tasks {
			if task == nil || task.UUID == "" {
				continue
			}
			result := tx.Omit("Run", "ParentTask", "InputArtifactsHydrated", "OutputArtifactsHydrated").Clauses(ignoreConflict()).Create(task)
			if result.Error != nil {
				return fmt.Errorf("persist task %q: %w", task.UUID, result.Error)
			}
			counts.Tasks++
			if result.RowsAffected == 1 {
				counts.InsertedTasks++
			} else {
				counts.ExistingTasks++
			}
		}
		// Parent task references are added after all task rows exist, allowing
		// arbitrary DAG ordering in the source snapshot.
		for _, task := range snapshot.Tasks {
			if task == nil || task.ParentTaskUUID == nil {
				continue
			}
			if err := tx.Model(&model.Task{}).Where("UUID = ?", task.UUID).Update("ParentTaskUUID", task.ParentTaskUUID).Error; err != nil {
				return fmt.Errorf("persist parent task for %q: %w", task.UUID, err)
			}
		}
		for _, artifact := range snapshot.Artifacts {
			if artifact == nil || artifact.UUID == "" {
				continue
			}
			result := tx.Omit("ArtifactTasks").Clauses(ignoreConflict()).Create(artifact)
			if result.Error != nil {
				return fmt.Errorf("persist artifact %q: %w", artifact.UUID, result.Error)
			}
			counts.Artifacts++
			if result.RowsAffected == 1 {
				counts.InsertedArtifacts++
			} else {
				counts.ExistingArtifacts++
			}
		}
		for _, relationship := range snapshot.Relationships {
			if relationship == nil || relationship.UUID == "" {
				continue
			}
			result := tx.Omit("Artifact", "Task", "Run").Clauses(ignoreConflict()).Create(relationship)
			if result.Error != nil {
				return fmt.Errorf("persist relationship %q: %w", relationship.UUID, result.Error)
			}
			counts.Relationships++
			if result.RowsAffected == 1 {
				counts.InsertedRelationships++
			} else {
				counts.ExistingRelationships++
			}
		}
		for _, metric := range snapshot.Metrics {
			if metric == nil || metric.RunUUID == "" || metric.NodeID == "" || metric.Name == "" {
				continue
			}
			result := tx.Clauses(ignoreConflict()).Create(metric)
			if result.Error != nil {
				return fmt.Errorf("persist metric %q for task %q: %w", metric.Name, metric.NodeID, result.Error)
			}
			counts.Metrics++
			if result.RowsAffected == 1 {
				counts.InsertedMetrics++
			} else {
				counts.ExistingMetrics++
			}
		}
		return validateSnapshotRows(tx, snapshot)
	})
	if err != nil {
		return Counts{}, err
	}
	return counts, nil
}

// Run executes the complete operator migration after the caller has opened the
// native database and MLMD source. A completed marker is a true no-op. A failed
// or interrupted run is restart-safe and idempotent, but currently re-reads the
// source inventory rather than resuming from a per-batch checkpoint.
func Run(ctx context.Context, db *gorm.DB, source MLMDSource, pageSize int32, toolVersion string, dryRun bool, progress ProgressFunc) (Counts, error) {
	if source == nil {
		return Counts{}, fmt.Errorf("MLMD source is required")
	}
	if progress == nil {
		progress = func(string, ...any) {}
	}
	var migrationLeaseToken string
	if !dryRun {
		leaseToken, completed, err := Start(db, toolVersion, time.Now())
		if err != nil {
			return Counts{}, err
		}
		if completed {
			var marker model.RuntimeMetadataMigration
			if err := db.Where("Name = ? AND Version = ?", model.RuntimeMetadataMigrationName, model.RuntimeMetadataMigrationVersion).First(&marker).Error; err != nil {
				return Counts{}, fmt.Errorf("read completed migration counts: %w", err)
			}
			var counts Counts
			if err := json.Unmarshal([]byte(marker.DestinationCounts), &counts); err != nil {
				return Counts{}, fmt.Errorf("decode completed migration counts: %w", err)
			}
			return counts, nil
		}
		migrationLeaseToken = leaseToken
	}
	snapshot, err := LoadSnapshot(ctx, source, pageSize)
	if err != nil {
		if !dryRun {
			if failureErr := Fail(db, migrationLeaseToken, err); failureErr != nil {
				return Counts{}, fmt.Errorf("migration failed: %w; additionally failed to record FAILED state: %v", err, failureErr)
			}
		}
		return Counts{}, err
	}
	progress("loaded MLMD snapshot: contexts=%d executions=%d artifacts=%d", len(snapshot.Contexts), len(snapshot.Executions), len(snapshot.Artifacts))
	converted, err := ConvertSnapshot(snapshot)
	if err != nil {
		if !dryRun {
			if failureErr := Fail(db, migrationLeaseToken, err); failureErr != nil {
				return Counts{}, fmt.Errorf("migration failed: %w; additionally failed to record FAILED state: %v", err, failureErr)
			}
		}
		return Counts{}, err
	}
	progress("converted native records: runs=%d tasks=%d artifacts=%d relationships=%d metrics=%d skipped=%d unsupported=%d", len(converted.Runs), len(converted.Tasks), len(converted.Artifacts), len(converted.Relationships), len(converted.Metrics), len(converted.Skipped), len(converted.Unsupported))
	for _, issue := range converted.Skipped {
		progress("skipped %s %d: %s", issue.Kind, issue.SourceID, issue.Reason)
	}
	for _, issue := range converted.Unsupported {
		progress("unsupported %s %d: %s", issue.Kind, issue.SourceID, issue.Reason)
	}
	if dryRun {
		return Counts{
			Runs: int64(len(converted.Runs)), Tasks: int64(len(converted.Tasks)),
			Artifacts: int64(len(converted.Artifacts)), Relationships: int64(len(converted.Relationships)), Metrics: int64(len(converted.Metrics)),
			Skipped: int64(len(converted.Skipped)), Unsupported: int64(len(converted.Unsupported)),
		}, nil
	}
	if len(converted.Skipped) != 0 || len(converted.Unsupported) != 0 {
		err := fmt.Errorf("migration found %d skipped and %d unsupported MLMD records; resolve them and rerun", len(converted.Skipped), len(converted.Unsupported))
		if failureErr := Fail(db, migrationLeaseToken, err); failureErr != nil {
			return Counts{}, fmt.Errorf("migration failed: %w; additionally failed to record FAILED state: %v", err, failureErr)
		}
		return Counts{}, err
	}
	counts, err := Persist(db, converted)
	if err != nil {
		if failureErr := Fail(db, migrationLeaseToken, err); failureErr != nil {
			return Counts{}, fmt.Errorf("migration failed: %w; additionally failed to record FAILED state: %v", err, failureErr)
		}
		return Counts{}, err
	}
	sourceCounts := map[string]int64{
		"contexts": int64(len(snapshot.Contexts)), "executions": int64(len(snapshot.Executions)),
		"artifacts": int64(len(snapshot.Artifacts)), "events": countEvents(snapshot),
	}
	destinationCounts := countsMap(counts)
	destinationCounts["skipped"] = int64(len(converted.Skipped))
	destinationCounts["unsupported"] = int64(len(converted.Unsupported))
	if err := Complete(db, migrationLeaseToken, sourceCounts, destinationCounts, toolVersion, time.Now(), func(tx *gorm.DB) error {
		return validateSnapshotRows(tx, converted)
	}); err != nil {
		if failureErr := Fail(db, migrationLeaseToken, err); failureErr != nil {
			return Counts{}, fmt.Errorf("migration failed: %w; additionally failed to record FAILED state: %v", err, failureErr)
		}
		return Counts{}, err
	}
	progress("migration completed: runs=%d tasks=%d artifacts=%d relationships=%d", counts.Runs, counts.Tasks, counts.Artifacts, counts.Relationships)
	return counts, nil
}

func countEvents(snapshot *Snapshot) int64 {
	var count int64
	for _, events := range snapshot.EventsByExecution {
		count += int64(len(events))
	}
	return count
}

func countsMap(counts Counts) map[string]int64 {
	return map[string]int64{
		"runs": counts.Runs, "tasks": counts.Tasks, "artifacts": counts.Artifacts,
		"relationships": counts.Relationships, "metrics": counts.Metrics,
		"inserted_runs": counts.InsertedRuns, "inserted_tasks": counts.InsertedTasks,
		"inserted_artifacts": counts.InsertedArtifacts, "inserted_relationships": counts.InsertedRelationships,
		"inserted_metrics": counts.InsertedMetrics, "existing_runs": counts.ExistingRuns,
		"existing_tasks": counts.ExistingTasks, "existing_artifacts": counts.ExistingArtifacts,
		"existing_relationships": counts.ExistingRelationships, "existing_metrics": counts.ExistingMetrics,
	}
}

func ignoreConflict() clause.OnConflict {
	return clause.OnConflict{DoNothing: true}
}

func validateSnapshotRows(tx *gorm.DB, snapshot *ConvertedSnapshot) error {
	for _, run := range snapshot.Runs {
		if run == nil {
			continue
		}
		var count int64
		if err := tx.Model(&model.Run{}).Where("UUID = ? AND Namespace = ?", run.UUID, run.Namespace).Count(&count).Error; err != nil || count != 1 {
			return fmt.Errorf("run %q failed namespace validation", run.UUID)
		}
		var stored model.Run
		if err := tx.Where("UUID = ?", run.UUID).First(&stored).Error; err != nil {
			return fmt.Errorf("run %q cannot be loaded for semantic validation: %w", run.UUID, err)
		}
		if stored.Namespace != run.Namespace || stored.DisplayName != run.DisplayName || stored.PipelineId != run.PipelineId || stored.PipelineVersionId != run.PipelineVersionId {
			return fmt.Errorf("run %q conflicts with existing native run data", run.UUID)
		}
	}
	for _, task := range snapshot.Tasks {
		if task == nil {
			continue
		}
		var count int64
		if err := tx.Model(&model.Task{}).Where("UUID = ? AND RunUUID = ? AND Namespace = ?", task.UUID, task.RunUUID, task.Namespace).Count(&count).Error; err != nil || count != 1 {
			return fmt.Errorf("task %q failed referential validation", task.UUID)
		}
		var stored model.Task
		if err := tx.Where("UUID = ?", task.UUID).First(&stored).Error; err != nil {
			return fmt.Errorf("task %q cannot be loaded for semantic validation: %w", task.UUID, err)
		}
		if stored.RunUUID != task.RunUUID || stored.Namespace != task.Namespace || stored.Name != task.Name || stored.Type != task.Type || stored.State != task.State || stored.Fingerprint != task.Fingerprint {
			return fmt.Errorf("task %q conflicts with existing native task data", task.UUID)
		}
		if task.Fingerprint != "" && stored.State != model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED) {
			return fmt.Errorf("task %q has a cache fingerprint but is not complete", task.UUID)
		}
		if task.ParentTaskUUID != nil {
			var parent model.Task
			if err := tx.Where("UUID = ?", *task.ParentTaskUUID).First(&parent).Error; err != nil {
				return fmt.Errorf("task %q parent %q is missing: %w", task.UUID, *task.ParentTaskUUID, err)
			}
			if parent.RunUUID != task.RunUUID || parent.Namespace != task.Namespace {
				return fmt.Errorf("task %q parent %q crosses run or namespace boundary", task.UUID, *task.ParentTaskUUID)
			}
		}
	}
	for _, artifact := range snapshot.Artifacts {
		if artifact == nil {
			continue
		}
		var count int64
		if err := tx.Model(&model.Artifact{}).Where("UUID = ? AND Namespace = ?", artifact.UUID, artifact.Namespace).Count(&count).Error; err != nil || count != 1 {
			return fmt.Errorf("artifact %q failed namespace validation", artifact.UUID)
		}
		var stored model.Artifact
		if err := tx.Where("UUID = ?", artifact.UUID).First(&stored).Error; err != nil {
			return fmt.Errorf("artifact %q cannot be loaded for semantic validation: %w", artifact.UUID, err)
		}
		if stored.Namespace != artifact.Namespace || stored.Name != artifact.Name || stored.Type != artifact.Type || !sameStringPtr(stored.URI, artifact.URI) {
			return fmt.Errorf("artifact %q conflicts with existing native artifact data", artifact.UUID)
		}
	}
	for _, relationship := range snapshot.Relationships {
		if relationship == nil {
			continue
		}
		var count int64
		if err := tx.Model(&model.ArtifactTask{}).Where("UUID = ?", relationship.UUID).Count(&count).Error; err != nil || count != 1 {
			return fmt.Errorf("relationship %q failed referential validation", relationship.UUID)
		}
		var task model.Task
		var artifact model.Artifact
		if err := tx.Where("UUID = ?", relationship.TaskID).First(&task).Error; err != nil {
			return fmt.Errorf("relationship %q task is missing: %w", relationship.UUID, err)
		}
		if err := tx.Where("UUID = ?", relationship.ArtifactID).First(&artifact).Error; err != nil {
			return fmt.Errorf("relationship %q artifact is missing: %w", relationship.UUID, err)
		}
		var stored model.ArtifactTask
		if err := tx.Where("UUID = ?", relationship.UUID).First(&stored).Error; err != nil {
			return fmt.Errorf("relationship %q cannot be loaded for semantic validation: %w", relationship.UUID, err)
		}
		if stored.ArtifactID != relationship.ArtifactID || stored.TaskID != relationship.TaskID || stored.RunUUID != relationship.RunUUID || stored.Type != relationship.Type || stored.Iteration != relationship.Iteration || stored.ArtifactKey != relationship.ArtifactKey {
			return fmt.Errorf("relationship %q conflicts with existing native relationship data", relationship.UUID)
		}
		if task.RunUUID != relationship.RunUUID || task.Namespace != artifact.Namespace {
			return fmt.Errorf("relationship %q crosses run or namespace boundary", relationship.UUID)
		}
		if !validIOType(relationship.Type) {
			return fmt.Errorf("relationship %q has unsupported I/O type %d", relationship.UUID, relationship.Type)
		}
		if relationship.Producer != nil {
			producerName, ok := relationship.Producer["taskName"].(string)
			if !ok || producerName == "" {
				return fmt.Errorf("relationship %q has invalid producer metadata", relationship.UUID)
			}
			var producer model.Task
			if err := tx.Where("RunUUID = ? AND Namespace = ? AND Name = ?", relationship.RunUUID, task.Namespace, producerName).First(&producer).Error; err != nil {
				return fmt.Errorf("relationship %q references missing producer task %q: %w", relationship.UUID, producerName, err)
			}
		}
	}
	for _, metric := range snapshot.Metrics {
		if metric == nil {
			continue
		}
		var run model.Run
		if err := tx.Where("UUID = ?", metric.RunUUID).First(&run).Error; err != nil {
			return fmt.Errorf("metric %q references missing run %q: %w", metric.Name, metric.RunUUID, err)
		}
		var task model.Task
		if err := tx.Where("RunUUID = ? AND Name = ?", metric.RunUUID, metric.NodeID).First(&task).Error; err != nil {
			return fmt.Errorf("metric %q references missing task %q: %w", metric.Name, metric.NodeID, err)
		}
		if task.Namespace != run.Namespace {
			return fmt.Errorf("metric %q crosses run namespace boundary", metric.Name)
		}
		var stored model.RunMetricV1
		if err := tx.Where("RunUUID = ? AND NodeID = ? AND Name = ?", metric.RunUUID, metric.NodeID, metric.Name).First(&stored).Error; err != nil {
			return fmt.Errorf("metric %q cannot be loaded for semantic validation: %w", metric.Name, err)
		}
		if stored.NumberValue != metric.NumberValue || stored.Format != metric.Format {
			return fmt.Errorf("metric %q conflicts with existing native metric data", metric.Name)
		}
	}
	return nil
}

func sameStringPtr(left, right *string) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func validIOType(ioType model.IOType) bool {
	switch apiv2beta1.IOType(ioType) {
	case apiv2beta1.IOType_COMPONENT_DEFAULT_INPUT, apiv2beta1.IOType_TASK_OUTPUT_INPUT,
		apiv2beta1.IOType_COMPONENT_INPUT, apiv2beta1.IOType_RUNTIME_VALUE_INPUT,
		apiv2beta1.IOType_ITERATOR_INPUT, apiv2beta1.IOType_ITERATOR_INPUT_RAW,
		apiv2beta1.IOType_ITERATOR_OUTPUT, apiv2beta1.IOType_OUTPUT,
		apiv2beta1.IOType_ONE_OF_OUTPUT,
		apiv2beta1.IOType_COLLECTED_INPUTS, apiv2beta1.IOType_TASK_FINAL_STATUS_OUTPUT:
		return true
	default:
		return false
	}
}
