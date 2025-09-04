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
// limitations under the License.

package storage

import (
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

const (
	testUUID1 = "123e4567-e89b-12d3-a456-426655441011"
	testUUID2 = "123e4567-e89b-12d3-a456-426655441012"
	testUUID3 = "123e4567-e89b-12d3-a456-426655441013"
)

// initializeTaskStore sets up a fake DB with a couple of runs and returns a TaskStore ready for testing.
func initializeTaskStore() (*DB, *TaskStore, *RunStore) {
	db := NewFakeDBOrFatal()
	fakeTime := util.NewFakeTimeForEpoch()
	// Seed a couple of runs to satisfy Task foreign key constraint.
	runStore := NewRunStore(db, fakeTime)
	run1 := &model.Run{
		UUID:         "run-1",
		ExperimentId: "exp-1",
		K8SName:      "run1",
		DisplayName:  "run1",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns1",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "Running",
			State:            model.RuntimeStateRunning,
		},
	}
	run2 := &model.Run{
		UUID:         "run-2",
		ExperimentId: "exp-2",
		K8SName:      "run2",
		DisplayName:  "run2",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns2",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "Succeeded",
			State:            model.RuntimeStateSucceeded,
		},
	}
	_, _ = runStore.CreateRun(run1)
	_, _ = runStore.CreateRun(run2)

	// Create task store with controllable UUID generator
	taskStore := NewTaskStore(db, fakeTime, util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil))
	return db, taskStore, runStore
}

// Minimal test to ensure model<->DB mapping remains valid.
func TestTaskAPIFieldMap(t *testing.T) {
	for _, modelField := range (&model.Task{}).APIToModelFieldMap() {
		assert.Contains(t, taskColumns, modelField)
	}
}

func TestCreateTask_Success(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	task := &model.Task{
		Namespace:        "ns1",
		PipelineName:     "pipeA",
		RunUUID:          "run-1",
		Pods:             model.JSONData{"pods": []interface{}{"pod-a", "pod-b"}},
		Fingerprint:      "fp-1",
		Name:             "taskA",
		ParentTaskUUID:   "",
		Status:           1,
		StateHistory:     model.JSONData(map[string]interface{}{"s": 1}),
		InputParameters:  model.JSONData(map[string]interface{}{"in": "x"}),
		OutputParameters: model.JSONData(map[string]interface{}{"out": "y"}),
		Type:             0,
		TypeAttrs:        model.JSONData(map[string]interface{}{"k": "v"}),
	}

	created, err := taskStore.CreateTask(task)
	assert.NoError(t, err)
	assert.Equal(t, testUUID1, created.UUID)
	// CreatedAt and StartedInSec should be auto-populated to the same timestamp (fake time starts from 0 -> 1)
	assert.Equal(t, created.CreatedAtInSec, created.StartedInSec)
	assert.Greater(t, created.CreatedAtInSec, int64(0))

	// Verify it can be fetched back
	fetched, err := taskStore.GetTask(created.UUID)
	assert.NoError(t, err)
	assert.Equal(t, created.UUID, fetched.UUID)
	assert.Equal(t, "ns1", fetched.Namespace)
	assert.Equal(t, "pipeA", fetched.PipelineName)
	assert.Equal(t, "run-1", fetched.RunUUID)
	assert.Equal(t, []interface{}{"pod-a", "pod-b"}, fetched.Pods["pods"])
	assert.Equal(t, "fp-1", fetched.Fingerprint)
	assert.Equal(t, "taskA", fetched.Name)
	assert.Equal(t, int32(1), fetched.Status)
	assert.Equal(t, int32(0), fetched.Type)
}

func TestGetTask_NotFound(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()
	_, err := taskStore.GetTask(testUUID1)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestListTasks_BasicAndFilters(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	// Create a parent task and two child tasks under different runs/pipelines
	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil)
	parent, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		PipelineName:     "pipeA",
		RunUUID:          "run-1",
		Pods:             model.JSONData{"pods": []interface{}{"p"}},
		Fingerprint:      "fp-parent",
		Status:           1,
		StateHistory:     map[string]interface{}{},
		InputParameters:  map[string]interface{}{},
		OutputParameters: map[string]interface{}{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID2, nil)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		PipelineName:     "pipeA",
		RunUUID:          "run-1",
		ParentTaskUUID:   parent.UUID,
		Pods:             model.JSONData{"pods": []interface{}{"c1"}},
		Fingerprint:      "fp-c1",
		Status:           1,
		StateHistory:     map[string]interface{}{},
		InputParameters:  map[string]interface{}{},
		OutputParameters: map[string]interface{}{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID3, nil)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:        "ns2",
		PipelineName:     "pipeB",
		RunUUID:          "run-2",
		ParentTaskUUID:   parent.UUID,
		Pods:             model.JSONData{"pods": []interface{}{"c2"}},
		Fingerprint:      "fp-c2",
		Status:           1,
		StateHistory:     map[string]interface{}{},
		InputParameters:  map[string]interface{}{},
		OutputParameters: map[string]interface{}{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	// List all tasks
	opts, _ := list.NewOptions(&model.Task{}, 10, "", nil)
	all, total, npt, err := taskStore.ListTasks(&model.FilterContext{}, opts)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(all))
	assert.Equal(t, 3, total)
	assert.Equal(t, "", npt)

	// Filter by RunUUID
	opts2, _ := list.NewOptions(&model.Task{}, 10, "", nil)
	runFiltered, total2, _, err := taskStore.ListTasks(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: "run-1"}}, opts2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(runFiltered))
	assert.Equal(t, 2, total2)

	// Filter by PipelineName
	opts3, _ := list.NewOptions(&model.Task{}, 10, "", nil)
	pipeFiltered, total3, _, err := taskStore.ListTasks(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.PipelineResourceType, ID: "pipeB"}}, opts3)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pipeFiltered))
	assert.Equal(t, 1, total3)

	// Filter by ParentTaskUUID (child tasks)
	opts4, _ := list.NewOptions(&model.Task{}, 10, "", nil)
	children, total4, _, err := taskStore.ListTasks(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.TaskResourceType, ID: parent.UUID}}, opts4)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(children))
	assert.Equal(t, 2, total4)
}

func TestUpdateTask_Success(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	// Create a task
	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil)
	created, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		PipelineName:     "pipeA",
		RunUUID:          "run-1",
		Pods:             model.JSONData{"pods": []interface{}{"p1"}},
		Fingerprint:      "fp-0",
		Status:           1,
		StateHistory:     map[string]interface{}{},
		InputParameters:  map[string]interface{}{},
		OutputParameters: map[string]interface{}{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	// Update some fields
	created.Name = "updatedName"
	created.Fingerprint = "fp-1"
	created.Pods = model.JSONData{"pods": []interface{}{"p2", "p3"}}
	created.Status = 2
	updated, err := taskStore.UpdateTask(created)
	assert.NoError(t, err)
	assert.Equal(t, created.UUID, updated.UUID)
	assert.Equal(t, "updatedName", updated.Name)
	assert.Equal(t, "fp-1", updated.Fingerprint)
	assert.Equal(t, []interface{}{"p2", "p3"}, updated.Pods["pods"])
	assert.Equal(t, int32(2), updated.Status)
}

func TestGetChildTasks_ReturnsChildren(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil)
	parent, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		PipelineName:     "pipeA",
		RunUUID:          "run-1",
		Pods:             model.JSONData{"pods": []interface{}{"p"}},
		Fingerprint:      "fp-p",
		Status:           1,
		StateHistory:     map[string]interface{}{},
		InputParameters:  map[string]interface{}{},
		OutputParameters: map[string]interface{}{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID2, nil)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		PipelineName:     "pipeA",
		RunUUID:          "run-1",
		ParentTaskUUID:   parent.UUID,
		Pods:             model.JSONData{"pods": []interface{}{"c1"}},
		Fingerprint:      "fp-a",
		Status:           1,
		StateHistory:     map[string]interface{}{},
		InputParameters:  model.JSONData(map[string]interface{}{"parameters": []interface{}{}, "artifacts": []interface{}{}, "metrics": []interface{}{}}),
		OutputParameters: model.JSONData(map[string]interface{}{"parameters": []interface{}{}, "artifacts": []interface{}{}, "metrics": []interface{}{}}),
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID3, nil)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		PipelineName:     "pipeA",
		RunUUID:          "run-1",
		ParentTaskUUID:   parent.UUID,
		Pods:             model.JSONData{"pods": []interface{}{"c2"}},
		Fingerprint:      "fp-b",
		Status:           1,
		StateHistory:     map[string]interface{}{},
		InputParameters:  model.JSONData(map[string]interface{}{"parameters": []interface{}{}, "artifacts": []interface{}{}, "metrics": []interface{}{}}),
		OutputParameters: model.JSONData(map[string]interface{}{"parameters": []interface{}{}, "artifacts": []interface{}{}, "metrics": []interface{}{}}),
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	children, err := taskStore.GetChildTasks(parent.UUID)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(children))
}
