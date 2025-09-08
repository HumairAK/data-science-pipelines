package storage

import (
	"fmt"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
)

const (
	linkUUID1   = "123e4567-e89b-12d3-a456-426655441011"
	linkUUID2   = "123e4567-e89b-12d3-a456-426655441012"
	artifactId1 = "123e4567-e89b-12d3-a456-426655441013"
	artifactId2 = "123e4567-e89b-12d3-a456-426655441014"
	taskId1     = "123e4567-e89b-12d3-a456-426655441015"
	taskId2     = "123e4567-e89b-12d3-a456-426655441016"
	runId1      = "123e4567-e89b-12d3-a456-426655441017"
	runId2      = "123e4567-e89b-12d3-a456-426655441018"
)

// initializeArtifactTaskDeps sets up a fake DB and returns stores needed for artifact-task tests.
func initializeArtifactTaskDeps() (*DB, *ArtifactStore, *TaskStore, *RunStore, *ArtifactTaskStore) {
	db := NewFakeDBOrFatal()
	fakeTime := util.NewFakeTimeForEpoch()

	artifactStore := NewArtifactStore(db, fakeTime, util.NewFakeUUIDGeneratorOrFatal(artifactId1, nil))
	taskStore := NewTaskStore(db, fakeTime, util.NewFakeUUIDGeneratorOrFatal(taskId1, nil))
	runStore := NewRunStore(db, fakeTime)
	linkStore := NewArtifactTaskStore(db, util.NewFakeUUIDGeneratorOrFatal(linkUUID1, nil))

	// Seed runs to satisfy Task FK
	_, _ = runStore.CreateRun(&model.Run{UUID: runId1, ExperimentId: "exp-1", K8SName: "r1", DisplayName: "r1", StorageState: model.StorageStateAvailable, Namespace: "ns1", RunDetails: model.RunDetails{CreatedAtInSec: 1, ScheduledAtInSec: 1, State: model.RuntimeStateRunning}})
	_, _ = runStore.CreateRun(&model.Run{UUID: runId2, ExperimentId: "exp-2", K8SName: "r2", DisplayName: "r2", StorageState: model.StorageStateAvailable, Namespace: "ns2", RunDetails: model.RunDetails{CreatedAtInSec: 2, ScheduledAtInSec: 2, State: model.RuntimeStateSucceeded}})

	return db, artifactStore, taskStore, runStore, linkStore
}

func TestArtifactTaskAPIFieldMap(t *testing.T) {
	for _, modelField := range (&model.ArtifactTask{}).APIToModelFieldMap() {
		assert.Contains(t, artifactTaskColumns, fmt.Sprintf("%s.%s", artifactTaskTableName, modelField))
	}
}

func TestCreateArtifactTask_Success(t *testing.T) {
	db, artifactStore, taskStore, _, linkStore := initializeArtifactTaskDeps()
	defer db.Close()

	// Create an artifact and a task to link
	artifactStore.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactId1, nil)
	art, err := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      1,
		Uri:       "s3://b/p1",
		Name:      "a1",
		Metadata:  map[string]interface{}{"k": "v"},
	})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(taskId1, nil)
	task, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		PipelineName:     "p1",
		RunUUID:          runId1,
		Name:             "t1",
		Pods:             model.JSONData{"pods": []interface{}{"p"}},
		Fingerprint:      "fp1",
		Status:           1,
		StateHistory:     map[string]interface{}{},
		InputParameters:  map[string]interface{}{},
		OutputParameters: map[string]interface{}{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	// Link as INPUT
	linkStore.uuid = util.NewFakeUUIDGeneratorOrFatal(linkUUID1, nil)
	link, err := linkStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID: art.UUID,
		TaskID:     task.UUID,
		Type:       apiv2beta1.ArtifactTaskType_INPUT,
	})
	assert.NoError(t, err)
	assert.Equal(t, linkUUID1, link.UUID)
	assert.Equal(t, art.UUID, link.ArtifactID)
	assert.Equal(t, task.UUID, link.TaskID)
	assert.Equal(t, apiv2beta1.ArtifactTaskType_INPUT, link.Type)

	// Fetch back
	got, err := linkStore.GetArtifactTask(link.UUID)
	assert.NoError(t, err)
	assert.Equal(t, link.UUID, got.UUID)
	assert.Equal(t, link.ArtifactID, got.ArtifactID)
	assert.Equal(t, link.TaskID, got.TaskID)
	assert.Equal(t, link.Type, got.Type)
}

func TestListArtifactTasks_Filters(t *testing.T) {
	db, artifactStore, taskStore, _, linkStore := initializeArtifactTaskDeps()
	defer db.Close()

	// Create 2 artifacts
	artifactStore.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactId1, nil)
	art1, err := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      1,
		Uri:       "u1",
		Name:      "a1",
		Metadata:  map[string]interface{}{},
	})
	assert.NoError(t, err)
	artifactStore.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactId2, nil)
	art2, err := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      1,
		Uri:       "u2",
		Name:      "a2",
		Metadata:  map[string]interface{}{},
	})
	assert.NoError(t, err)

	// Create 2 tasks across 2 runs
	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(taskId1, nil)
	t1, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		PipelineName:     "p1",
		RunUUID:          runId1,
		Name:             "t1",
		Pods:             model.JSONData{"pods": []interface{}{"p1"}},
		Fingerprint:      "fp-1",
		Status:           1,
		StateHistory:     map[string]interface{}{},
		InputParameters:  map[string]interface{}{},
		OutputParameters: map[string]interface{}{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)
	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(taskId2, nil)
	t2, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns2",
		PipelineName:     "p2",
		RunUUID:          runId2,
		Name:             "t2",
		Pods:             model.JSONData{"pods": []interface{}{"p2"}},
		Fingerprint:      "fp-2",
		Status:           1,
		StateHistory:     map[string]interface{}{},
		InputParameters:  map[string]interface{}{},
		OutputParameters: map[string]interface{}{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	// Create links: art1<->t1 (INPUT), art2<->t1 (OUTPUT), art2<->t2 (INPUT)
	linkStore.uuid = util.NewFakeUUIDGeneratorOrFatal(linkUUID1, nil)
	_, err = linkStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID: art1.UUID,
		TaskID:     t1.UUID,
		Type:       apiv2beta1.ArtifactTaskType_INPUT,
	})
	assert.NoError(t, err)
	linkStore.uuid = util.NewFakeUUIDGeneratorOrFatal(linkUUID2, nil)
	_, err = linkStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID: art2.UUID,
		TaskID:     t1.UUID,
		Type:       apiv2beta1.ArtifactTaskType_OUTPUT,
	})
	assert.NoError(t, err)
	// another link with a fresh random UUID
	linkStore.uuid = util.NewUUIDGenerator()
	_, err = linkStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID: art2.UUID,
		TaskID:     t2.UUID,
		Type:       apiv2beta1.ArtifactTaskType_INPUT,
	})
	assert.NoError(t, err)

	opts, _ := list.NewOptions(&model.ArtifactTask{}, 20, "", nil)

	// List all
	all, total, npt, err := linkStore.ListArtifactTasks(nil, opts)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(all))
	assert.Equal(t, 3, total)
	assert.Equal(t, "", npt)

	// Filter by task t1
	byTask, totalTask, _, err := linkStore.ListArtifactTasks([]*model.FilterContext{{ReferenceKey: &model.ReferenceKey{Type: model.TaskResourceType, ID: t1.UUID}}}, opts)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(byTask))
	assert.Equal(t, 2, totalTask)

	// Filter by artifact art2
	byArtifact, totalArt, _, err := linkStore.ListArtifactTasks([]*model.FilterContext{{ReferenceKey: &model.ReferenceKey{Type: model.ArtifactResourceType, ID: art2.UUID}}}, opts)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(byArtifact)) // art2 is linked twice
	assert.Equal(t, 2, totalArt)

	// Filter by run runId2 (should return only links for tasks in run-2)
	byRun, totalRun, _, err := linkStore.ListArtifactTasks([]*model.FilterContext{{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: runId2}}}, opts)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(byRun))
	assert.Equal(t, 1, totalRun)
	assert.Equal(t, art2.UUID, byRun[0].ArtifactID)
	assert.Equal(t, t2.UUID, byRun[0].TaskID)
	assert.Equal(t, apiv2beta1.ArtifactTaskType_INPUT, byRun[0].Type)
}

func TestListArtifactsForTask_UsingArtifactTasks(t *testing.T) {
	db, artifactStore, taskStore, _, linkStore := initializeArtifactTaskDeps()
	defer db.Close()

	// Seed artifacts and a single task
	artifactStore.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactId1, nil)
	art1, err := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      1,
		Uri:       "u1",
		Name:      "a1",
		Metadata:  map[string]interface{}{},
	})
	assert.NoError(t, err)
	artifactStore.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactId2, nil)
	art2, err := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      1,
		Uri:       "u2",
		Name:      "a2",
		Metadata:  map[string]interface{}{},
	})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(taskId1, nil)
	t1, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		PipelineName:     "p1",
		RunUUID:          runId1,
		Name:             "t1",
		Pods:             model.JSONData{"pods": []interface{}{"p1"}},
		Fingerprint:      "fp-1",
		Status:           1,
		StateHistory:     map[string]interface{}{},
		InputParameters:  map[string]interface{}{},
		OutputParameters: map[string]interface{}{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	// Link both artifacts to t1
	linkStore.uuid = util.NewFakeUUIDGeneratorOrFatal(linkUUID1, nil)
	_, err = linkStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID: art1.UUID,
		TaskID:     t1.UUID,
		Type:       apiv2beta1.ArtifactTaskType_INPUT,
	})
	assert.NoError(t, err)
	linkStore.uuid = util.NewFakeUUIDGeneratorOrFatal(linkUUID2, nil)
	_, err = linkStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID: art2.UUID,
		TaskID:     t1.UUID,
		Type:       apiv2beta1.ArtifactTaskType_OUTPUT,
	})
	assert.NoError(t, err)

	// Use artifactTasks to list artifacts for task t1
	opts, _ := list.NewOptions(&model.ArtifactTask{}, 20, "", nil)
	rows, total, _, err := linkStore.ListArtifactTasks([]*model.FilterContext{{ReferenceKey: &model.ReferenceKey{Type: model.TaskResourceType, ID: t1.UUID}}}, opts)
	assert.NoError(t, err)
	assert.Equal(t, 2, total)

	// Collect artifact IDs and verify set equals {art1, art2}
	ids := map[string]bool{}
	for _, r := range rows {
		ids[r.ArtifactID] = true
	}
	assert.True(t, ids[art1.UUID])
	assert.True(t, ids[art2.UUID])
	assert.Equal(t, 2, len(ids))
}
