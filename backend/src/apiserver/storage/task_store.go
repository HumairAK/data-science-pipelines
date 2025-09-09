// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.package storage

package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

const tableName = "tasks"

var taskColumns = []string{
	"UUID",
	"Namespace",
	"PipelineName",
	"RunUUID",
	"Pods",
	"CreatedAtInSec",
	"StartedInSec",
	"FinishedInSec",
	"Fingerprint",
	"Name",
	"DisplayName",
	"ParentTaskUUID",
	"Status",
	"StatusMetadata",
	"StateHistory",
	"InputParameters",
	"InputArtifacts",
	"OutputParameters",
	"OutputArtifacts",
	"Type",
	"TypeAttrs",
}

var taskColumnsWithPayload = append(taskColumns, "Payload")

// Ensure TaskStore implements TaskStoreInterface
var _ TaskStoreInterface = (*TaskStore)(nil)

type TaskStoreInterface interface {
	// CreateTask Create a task entry in the database.
	CreateTask(task *model.Task) (*model.Task, error)

	// GetTask Fetches a task with a given id.
	GetTask(id string) (*model.Task, error)

	// ListTasks Fetches tasks for given filtering and listing options.
	ListTasks(filterContext *model.FilterContext, opts *list.Options) ([]*model.Task, int, string, error)

	// UpdateTask Updates an existing task entry in the database.
	UpdateTask(task *model.Task) (*model.Task, error)

	// GetChildTasks Fetches all child tasks for a given task UUID.
	GetChildTasks(taskId string) ([]*model.Task, error)
}

type TaskStore struct {
	db   *DB
	time util.TimeInterface
	uuid util.UUIDGeneratorInterface
}

// NewTaskStore creates a new TaskStore.
func NewTaskStore(db *DB, time util.TimeInterface, uuid util.UUIDGeneratorInterface) *TaskStore {
	return &TaskStore{
		db:   db,
		time: time,
		uuid: uuid,
	}
}

// scanTaskRow scans a single row into a model.Task. It expects the column order to match taskColumns.
func scanTaskRow(rowscanner interface{ Scan(dest ...any) error }) (*model.Task, error) {
	var uuid, namespace, pipelineName, runUUID, fingerprint string
	var name, displayName, parentTaskId, pods, statusMetadata, stateHistory, inputParams, inputArtifacts, outputParams, outputArtifacts, typeAttrs sql.NullString
	var createdAtInSec, startedInSec, finishedInSec sql.NullInt64
	var taskStatus, taskType int32
	if err := rowscanner.Scan(
		&uuid,
		&namespace,
		&pipelineName,
		&runUUID,
		&pods,
		&createdAtInSec,
		&startedInSec,
		&finishedInSec,
		&fingerprint,
		&name,
		&displayName,
		&parentTaskId,
		&taskStatus,
		&statusMetadata,
		&stateHistory,
		&inputParams,
		&inputArtifacts,
		&outputParams,
		&outputArtifacts,
		&taskType,
		&typeAttrs,
	); err != nil {
		return nil, err
	}
	var statusMetadataNew model.JSONData
	if statusMetadata.Valid {
		if err := json.Unmarshal([]byte(statusMetadata.String), &statusMetadataNew); err != nil {
			return nil, err
		}
	}
	var stateHistoryNew model.JSONSlice
	if stateHistory.Valid {
		if err := json.Unmarshal([]byte(stateHistory.String), &stateHistoryNew); err != nil {
			return nil, err
		}
	}
	var podsNew model.JSONSlice
	if pods.Valid {
		if err := json.Unmarshal([]byte(pods.String), &podsNew); err != nil {
			return nil, err
		}
	}
	var inputParameters model.JSONSlice
	if inputParams.Valid {
		if err := json.Unmarshal([]byte(inputParams.String), &inputParameters); err != nil {
			return nil, err
		}
	}
	var outputParameters model.JSONSlice
	if outputParams.Valid {
		if err := json.Unmarshal([]byte(outputParams.String), &outputParameters); err != nil {
			return nil, err
		}
	}
	var inputArtifactsData model.JSONSlice
	if inputArtifacts.Valid {
		if err := json.Unmarshal([]byte(inputArtifacts.String), &inputArtifactsData); err != nil {
			return nil, err
		}
	}
	var outputArtifactsData model.JSONSlice
	if outputArtifacts.Valid {
		if err := json.Unmarshal([]byte(outputArtifacts.String), &outputArtifactsData); err != nil {
			return nil, err
		}
	}
	var typeAttrsData model.JSONData
	if typeAttrs.Valid {
		if err := json.Unmarshal([]byte(typeAttrs.String), &typeAttrsData); err != nil {
			return nil, err
		}
	}
	return &model.Task{
		UUID:             uuid,
		Namespace:        namespace,
		PipelineName:     pipelineName,
		RunUUID:          runUUID,
		Pods:             podsNew,
		CreatedAtInSec:   createdAtInSec.Int64,
		StartedInSec:     startedInSec.Int64,
		FinishedInSec:    finishedInSec.Int64,
		Fingerprint:      fingerprint,
		Name:             name.String,
		DisplayName:      displayName.String,
		ParentTaskUUID:   parentTaskId.String,
		Status:           taskStatus,
		StatusMetadata:   statusMetadataNew,
		StateHistory:     stateHistoryNew,
		InputParameters:  inputParameters,
		InputArtifacts:   inputArtifactsData,
		OutputParameters: outputParameters,
		OutputArtifacts:  outputArtifactsData,
		Type:             taskType,
		TypeAttrs:        typeAttrsData,
	}, nil
}

func (s *TaskStore) scanRows(rows *sql.Rows) ([]*model.Task, error) {
	var tasks []*model.Task
	for rows.Next() {
		t, err := scanTaskRow(rows)
		if err != nil {
			fmt.Printf("scan error is %v", err)
			return tasks, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (s *TaskStore) CreateTask(task *model.Task) (*model.Task, error) {
	// Set up UUID for task.
	newTask := *task
	id, err := s.uuid.NewRandom()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create an task id")
	}
	newTask.UUID = id.String()

	if newTask.CreatedAtInSec == 0 {
		if newTask.StartedInSec == 0 {
			now := s.time.Now().Unix()
			newTask.StartedInSec = now
			newTask.CreatedAtInSec = now
		} else {
			newTask.CreatedAtInSec = newTask.StartedInSec
		}
	}

	stateHistoryString := ""
	if history, err := json.Marshal(newTask.StateHistory); err == nil {
		stateHistoryString = string(history)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal state history in a new run")
	}

	podsString := ""
	if podNames, err := json.Marshal(newTask.Pods); err == nil {
		podsString = string(podNames)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal pod names in a new task")
	}

	inputParamsString := ""
	if inputParams, err := json.Marshal(newTask.InputParameters); err == nil {
		inputParamsString = string(inputParams)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal input parameters in a new task")
	}

	inputArtifactsString := ""
	if ia, err := json.Marshal(newTask.InputArtifacts); err == nil {
		inputArtifactsString = string(ia)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal input artifacts in a new task")
	}

	outputParamsString := ""
	if outputParams, err := json.Marshal(newTask.OutputParameters); err == nil {
		outputParamsString = string(outputParams)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal output parameters in a new task")
	}

	outputArtifactsString := ""
	if oa, err := json.Marshal(newTask.OutputArtifacts); err == nil {
		outputArtifactsString = string(oa)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal output artifacts in a new task")
	}

	typeAttrsString := ""
	if typeAttrs, err := json.Marshal(newTask.TypeAttrs); err == nil {
		typeAttrsString = string(typeAttrs)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal type attributes in a new task")
	}

	sql, args, err := sq.
		Insert(tableName).
		SetMap(
			sq.Eq{
				"UUID":             newTask.UUID,
				"Namespace":        newTask.Namespace,
				"PipelineName":     newTask.PipelineName,
				"RunUUID":          newTask.RunUUID,
				"Pods":             podsString,
				"CreatedAtInSec":   newTask.CreatedAtInSec,
				"StartedInSec":     newTask.StartedInSec,
				"FinishedInSec":    newTask.FinishedInSec,
				"Fingerprint":      newTask.Fingerprint,
				"Name":             newTask.Name,
				"ParentTaskUUID":   newTask.ParentTaskUUID,
				"Status":           newTask.Status,
				"StateHistory":     stateHistoryString,
				"InputParameters":  inputParamsString,
				"InputArtifacts":   inputArtifactsString,
				"OutputParameters": outputParamsString,
				"OutputArtifacts":  outputArtifactsString,
				"Type":             newTask.Type,
				"TypeAttrs":        typeAttrsString,
			},
		).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to insert task to task table: %v",
			err.Error())
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to add task to task table: %v",
			err.Error())
	}
	return &newTask, nil
}

// ListTasks Runs two SQL queries in a transaction to return a list of matching experiments, as well as their
// total_size. The total_size does not reflect the page size.
func (s *TaskStore) ListTasks(filterContext *model.FilterContext, opts *list.Options) ([]*model.Task, int, string, error) {
	errorF := func(err error) ([]*model.Task, int, string, error) {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to list tasks: %v", err)
	}

	// SQL for getting the filtered and paginated rows
	sqlBuilder := sq.Select(taskColumns...).From("tasks")
	if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.PipelineResourceType {
		sqlBuilder = sqlBuilder.Where(sq.Eq{"PipelineName": filterContext.ReferenceKey.ID})
	}
	if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.RunResourceType {
		sqlBuilder = sqlBuilder.Where(sq.Eq{"RunUUID": filterContext.ReferenceKey.ID})
	}
	if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.TaskResourceType {
		sqlBuilder = sqlBuilder.Where(sq.Eq{"ParentTaskUUID": filterContext.ReferenceKey.ID})
	}
	sqlBuilder = opts.AddFilterToSelect(sqlBuilder)

	rowsSql, rowsArgs, err := opts.AddPaginationToSelect(sqlBuilder).ToSql()
	if err != nil {
		return errorF(err)
	}

	// SQL for getting total size. This matches the query to get all the rows above, in order
	// to do the same filter, but counts instead of scanning the rows.
	sqlBuilder = sq.Select("count(*)").From("tasks")
	if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.PipelineResourceType {
		sqlBuilder = sqlBuilder.Where(sq.Eq{"PipelineName": filterContext.ReferenceKey.ID})
	}
	if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.RunResourceType {
		sqlBuilder = sqlBuilder.Where(sq.Eq{"RunUUID": filterContext.ReferenceKey.ID})
	}
	if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.TaskResourceType {
		sqlBuilder = sqlBuilder.Where(sq.Eq{"ParentTaskUUID": filterContext.ReferenceKey.ID})
	}
	sizeSql, sizeArgs, err := opts.AddFilterToSelect(sqlBuilder).ToSql()
	if err != nil {
		return errorF(err)
	}

	// Use a transaction to make sure we're returning the total_size of the same rows queried
	tx, err := s.db.Begin()
	if err != nil {
		glog.Errorf("Failed to start transaction to list tasks")
		return errorF(err)
	}

	rows, err := tx.Query(rowsSql, rowsArgs...)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	if err := rows.Err(); err != nil {
		tx.Rollback()
		return errorF(err)
	}
	exps, err := s.scanRows(rows)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	defer rows.Close()

	sizeRow, err := tx.Query(sizeSql, sizeArgs...)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	if err := sizeRow.Err(); err != nil {
		tx.Rollback()
		return errorF(err)
	}
	total_size, err := list.ScanRowToTotalSize(sizeRow)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	defer sizeRow.Close()

	err = tx.Commit()
	if err != nil {
		glog.Errorf("Failed to commit transaction to list experiments")
		return errorF(err)
	}

	if len(exps) <= opts.PageSize {
		return exps, total_size, "", nil
	}

	npt, err := opts.NextPageToken(exps[opts.PageSize])
	return exps[:opts.PageSize], total_size, npt, err
}

func (s *TaskStore) GetTask(id string) (*model.Task, error) {
	toSql, args, err := sq.
		Select(taskColumns...).
		From("tasks").
		Where(sq.Eq{"tasks.uuid": id}).
		Limit(1).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get task: %v", err.Error())
	}
	r, err := s.db.Query(toSql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get task: %v", err.Error())
	}
	defer r.Close()
	tasks, err := s.scanRows(r)

	if err != nil || len(tasks) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to get pipeline: %v", err)
	}
	if len(tasks) == 0 {
		return nil, util.NewResourceNotFoundError("task", fmt.Sprint(id))
	}
	return tasks[0], nil
}

// UpdateTask updates an existing task in the tasks table and returns the updated task.
func (s *TaskStore) UpdateTask(task *model.Task) (*model.Task, error) {
	if task == nil {
		return nil, util.NewInvalidInputError("Failed to update task: task cannot be nil")
	}
	if task.UUID == "" {
		return nil, util.NewInvalidInputError("Failed to update task: task ID cannot be empty")
	}

	// Marshal complex fields to JSON strings where needed
	stateHistoryString := ""
	if history, err := json.Marshal(task.StateHistory); err == nil {
		stateHistoryString = string(history)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal state history in an updated task")
	}

	statusMetadataString := ""
	if statusMetadata, err := json.Marshal(task.StatusMetadata); err == nil {
		statusMetadataString = string(statusMetadata)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal status metadata in an updated task")
	}

	podsString := ""
	if pods, err := json.Marshal(task.Pods); err == nil {
		podsString = string(pods)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal pod names in an updated task")
	}

	inputParamsString := ""
	if inputParams, err := json.Marshal(task.InputParameters); err == nil {
		inputParamsString = string(inputParams)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal input parameters in an updated task")
	}

	inputArtifactsString := ""
	if ia, err := json.Marshal(task.InputArtifacts); err == nil {
		inputArtifactsString = string(ia)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal input artifacts in an updated task")
	}

	outputParamsString := ""
	if outputParams, err := json.Marshal(task.OutputParameters); err == nil {
		outputParamsString = string(outputParams)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal output parameters in an updated task")
	}

	outputArtifactsString := ""
	if oa, err := json.Marshal(task.OutputArtifacts); err == nil {
		outputArtifactsString = string(oa)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal output artifacts in an updated task")
	}

	typeAttrsString := ""
	if typeAttrs, err := json.Marshal(task.TypeAttrs); err == nil {
		typeAttrsString = string(typeAttrs)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal type attributes in an updated task")
	}

	// Build the update statement. We do not update CreatedAtInSec.
	sqlStr, args, err := sq.
		Update(tableName).
		SetMap(sq.Eq{
			"Namespace":        task.Namespace,
			"PipelineName":     task.PipelineName,
			"RunUUID":          task.RunUUID,
			"Pods":             podsString,
			"StartedInSec":     task.StartedInSec,
			"FinishedInSec":    task.FinishedInSec,
			"Fingerprint":      task.Fingerprint,
			"Name":             task.Name,
			"ParentTaskUUID":   task.ParentTaskUUID,
			"Status":           task.Status,
			"StatusMetadata":   statusMetadataString,
			"StateHistory":     stateHistoryString,
			"InputParameters":  inputParamsString,
			"InputArtifacts":   inputArtifactsString,
			"OutputParameters": outputParamsString,
			"OutputArtifacts":  outputArtifactsString,
			"Type":             task.Type,
			"TypeAttrs":        typeAttrsString,
		}).
		Where(sq.Eq{"UUID": task.UUID}).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to update task: %v", err.Error())
	}

	res, err := s.db.Exec(sqlStr, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to update task: %v", err.Error())
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return nil, util.NewResourceNotFoundError("task", task.UUID)
	}

	return s.GetTask(task.UUID)
}

func (s *TaskStore) GetChildTasks(taskId string) ([]*model.Task, error) {
	toSql, args, err := sq.
		Select(taskColumns...).
		From("tasks").
		Where(sq.Eq{"ParentTaskUUID": taskId}).
		ToSql()

	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get child tasks: %v", err.Error())
	}

	rows, err := s.db.Query(toSql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get child tasks: %v", err.Error())
	}
	defer rows.Close()

	return s.scanRows(rows)
}
