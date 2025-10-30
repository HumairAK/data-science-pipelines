package server

import (
	"context"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

// Helper to create a simple run via resource manager and return its ID.
func seedOneRun(t *testing.T) (*resource.FakeClientManager, *resource.ResourceManager, string) {
	clients, manager, run := initWithOneTimeRunV2(t)
	return clients, manager, run.UUID
}

func TestTask_Create_Update_Get_List(t *testing.T) {
	// Single-user mode by default; keep it to bypass authz.
	clients, manager, runID := seedOneRun(t)
	defer clients.Close()

	runSrv := createRunServer(manager)

	// Create task with inputs/outputs
	v1, err := structpb.NewValue("v1")
	assert.NoError(t, err)
	v2, err := structpb.NewValue("3.14")
	assert.NoError(t, err)
	inParams := []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
		{
			Value:        v1,
			ParameterKey: "p1",
		},
	}
	outParams := []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
		{
			Value:        v2,
			ParameterKey: "op1",
		},
	}
	createReq := &apiv2beta1.CreateTaskRequest{Task: &apiv2beta1.PipelineTaskDetail{
		RunId:   runID,
		Name:    "trainer",
		State:   apiv2beta1.PipelineTaskDetail_RUNNING,
		Type:    apiv2beta1.PipelineTaskDetail_RUNTIME,
		Inputs:  &apiv2beta1.PipelineTaskDetail_InputOutputs{Parameters: inParams},
		Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{Parameters: outParams},
	}}
	created, err := runSrv.CreateTask(context.Background(), createReq)
	assert.NoError(t, err)
	assert.NotEmpty(t, created.GetTaskId())
	assert.Equal(t, runID, created.GetRunId())
	assert.Equal(t, "trainer", created.GetName())
	// Verify inputs/outputs echoed back
	assert.Len(t, created.GetInputs().GetParameters(), 1)
	assert.Equal(t, "p1", created.GetInputs().GetParameters()[0].GetParameterKey())
	assert.Len(t, created.GetOutputs().GetParameters(), 1)
	assert.Equal(t, "op1", created.GetOutputs().GetParameters()[0].GetParameterKey())

	// Update task: change status and outputs
	updReq := &apiv2beta1.UpdateTaskRequest{TaskId: created.GetTaskId(), Task: &apiv2beta1.PipelineTaskDetail{
		TaskId: created.GetTaskId(),
		RunId:  runID,
		Name:   "trainer",
		State:  apiv2beta1.PipelineTaskDetail_SUCCEEDED,
		Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{Parameters: []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
			{
				Value:        func() *structpb.Value { v, _ := structpb.NewValue("done"); return v }(),
				ParameterKey: "op1",
			},
		}},
	}}
	updated, err := runSrv.UpdateTask(context.Background(), updReq)
	assert.NoError(t, err)
	assert.Equal(t, apiv2beta1.PipelineTaskDetail_SUCCEEDED, updated.GetState())
	assert.Equal(t, "done", updated.GetOutputs().GetParameters()[0].GetValue().AsInterface())

	// GetTask
	got, err := runSrv.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: created.GetTaskId()})
	assert.NoError(t, err)
	assert.Equal(t, created.GetTaskId(), got.GetTaskId())
	assert.Equal(t, apiv2beta1.PipelineTaskDetail_SUCCEEDED, got.GetState())

	// ListTasks by run ID
	listResp, err := runSrv.ListTasks(context.Background(), &apiv2beta1.ListTasksRequest{ParentFilter: &apiv2beta1.ListTasksRequest_RunId{RunId: runID}, PageSize: 50})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, int(listResp.GetTotalSize()), 1)
	found := false
	for _, tt := range listResp.GetTasks() {
		if tt.GetTaskId() == created.GetTaskId() {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestTask_RunHydration_WithInputsOutputs_ArtifactsAndMetrics(t *testing.T) {
	// Multi-user on to exercise auth paths, but use helper ctx for headers.
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })

	clients, manager, run := initWithOneTimeRun(t)
	defer clients.Close()

	runSrv := createRunServer(manager)
	artSrv := createArtifactServer(manager)

	// Create a task with IO
	create := &apiv2beta1.CreateTaskRequest{
		Task: &apiv2beta1.PipelineTaskDetail{
			RunId: run.UUID,
			Name:  "preprocess",
			State: apiv2beta1.PipelineTaskDetail_RUNNING,
			Inputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{Parameters: []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
				{
					Value:        func() *structpb.Value { v, _ := structpb.NewValue("0.5"); return v }(),
					ParameterKey: "threshold",
				},
			}},
			Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{Parameters: []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
				{
					Value:        func() *structpb.Value { v, _ := structpb.NewValue("100"); return v }(),
					ParameterKey: "rows",
				},
			}},
		}}
	created, err := runSrv.CreateTask(ctxWithUser(), create)
	assert.NoError(t, err)

	// Create an artifact and link it as output of the task
	_, err = artSrv.CreateArtifact(ctxWithUser(),
		&apiv2beta1.CreateArtifactRequest{
			RunId:       run.UUID,
			TaskId:      created.GetTaskId(),
			ProducerKey: "some-parent-task-output",
			Type:        apiv2beta1.IOType_TASK_OUTPUT_INPUT,
			Artifact: &apiv2beta1.Artifact{
				Namespace: run.Namespace,
				Type:      apiv2beta1.Artifact_Model,
				Uri:       strPTR("gs://bucket/model"),
				Name:      "m1",
			}})
	assert.NoError(t, err)

	// Confirm a link was created between the task and the artifact
	artifactTasks, err := artSrv.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{
		TaskIds:  []string{created.GetTaskId()},
		RunIds:   []string{run.UUID},
		PageSize: 10,
	})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), artifactTasks.GetTotalSize())
	assert.Equal(t, 1, len(artifactTasks.GetArtifactTasks()))

	// Update task outputs to include an artifact reference in OutputArtifacts
	_, err = runSrv.UpdateTask(ctxWithUser(),
		&apiv2beta1.UpdateTaskRequest{
			TaskId: created.GetTaskId(),
			Task: &apiv2beta1.PipelineTaskDetail{
				TaskId:  created.GetTaskId(),
				RunId:   run.UUID,
				State:   apiv2beta1.PipelineTaskDetail_SUCCEEDED,
				Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{},
			}})
	assert.NoError(t, err)

	// Now fetch the run and ensure tasks are hydrated with inputs/outputs
	gr, err := runSrv.GetRun(ctxWithUser(), &apiv2beta1.GetRunRequest{RunId: run.UUID})
	assert.NoError(t, err)
	assert.NotNil(t, gr)
	assert.GreaterOrEqual(t, len(gr.GetTasks()), 1)
	var taskFound *apiv2beta1.PipelineTaskDetail
	for _, tt := range gr.GetTasks() {
		if tt.GetTaskId() == created.GetTaskId() {
			taskFound = tt
			break
		}
	}
	if assert.NotNil(t, taskFound, "created task not present in hydrated run") {
		// Parameters present
		assert.Equal(t, 1, len(taskFound.GetInputs().GetParameters()))
		if assert.NotNil(t, taskFound.GetInputs().GetParameters(), "parameters not present in hydrated task") {
			assert.Equal(t, "threshold", taskFound.GetInputs().GetParameters()[0].GetParameterKey())
			// Outputs updated and artifact reference present
			assert.Equal(t, apiv2beta1.PipelineTaskDetail_SUCCEEDED, taskFound.GetState())
		}
		assert.Equal(t, 1, len(taskFound.GetOutputs().GetArtifacts()))
		if assert.NotNil(t, taskFound.GetOutputs().GetArtifacts(), "artifacts not present in hydrated task") {
			ioArtifact := taskFound.GetOutputs().GetArtifacts()[0]
			assert.Equal(t, "some-parent-task-output", ioArtifact.GetArtifactKey())
			if assert.NotNil(t, ioArtifact.GetProducer()) {
				assert.Equal(t, taskFound.Name, ioArtifact.GetProducer().GetTaskName())
			}
			if len(ioArtifact.GetArtifacts()) > 0 {
				artifact := ioArtifact.GetArtifacts()[0]
				assert.Equal(t, "gs://bucket/model", *artifact.Uri)
				assert.Equal(t, "m1", artifact.Name)
			}
		}
	}

}

func TestListTasks_ByParent(t *testing.T) {
	cm, rm, runID := seedOneRun(t)
	defer cm.Close()
	server := createRunServer(rm)

	// Create parent task
	parent, err := server.CreateTask(context.Background(),
		&apiv2beta1.CreateTaskRequest{
			Task: &apiv2beta1.PipelineTaskDetail{
				RunId: runID,
				Name:  "parent",
			},
		},
	)
	assert.NoError(t, err)

	// Create child task with ParentTaskId
	child, err := server.CreateTask(context.Background(),
		&apiv2beta1.CreateTaskRequest{
			Task: &apiv2beta1.PipelineTaskDetail{
				RunId:        runID,
				Name:         "child",
				ParentTaskId: strPTR(parent.GetTaskId()),
			},
		},
	)
	assert.NoError(t, err)
	assert.NotEmpty(t, child.GetTaskId())

	// List by parent ID
	resp, err := server.ListTasks(context.Background(),
		&apiv2beta1.ListTasksRequest{
			ParentFilter: &apiv2beta1.ListTasksRequest_ParentId{
				ParentId: parent.GetTaskId(),
			},
			PageSize: 50,
		})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), resp.GetTotalSize())
	assert.Equal(t, 1, len(resp.GetTasks()))
	assert.Equal(t, child.GetTaskId(), resp.GetTasks()[0].GetTaskId())
}

func TestUpdateTasksBulk_Success(t *testing.T) {
	// Single-user mode to bypass authz
	clients, manager, runID := seedOneRun(t)
	defer clients.Close()

	runSrv := createRunServer(manager)

	// Create three tasks
	v1, _ := structpb.NewValue("initial1")
	v2, _ := structpb.NewValue("initial2")
	v3, _ := structpb.NewValue("initial3")

	task1, err := runSrv.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		Task: &apiv2beta1.PipelineTaskDetail{
			RunId: runID,
			Name:  "task1",
			State: apiv2beta1.PipelineTaskDetail_RUNNING,
			Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{
				Parameters: []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
					{Value: v1, ParameterKey: "out1"},
				},
			},
		},
	})
	assert.NoError(t, err)

	task2, err := runSrv.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		Task: &apiv2beta1.PipelineTaskDetail{
			RunId: runID,
			Name:  "task2",
			State: apiv2beta1.PipelineTaskDetail_RUNNING,
			Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{
				Parameters: []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
					{Value: v2, ParameterKey: "out2"},
				},
			},
		},
	})
	assert.NoError(t, err)

	task3, err := runSrv.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		Task: &apiv2beta1.PipelineTaskDetail{
			RunId: runID,
			Name:  "task3",
			State: apiv2beta1.PipelineTaskDetail_RUNNING,
			Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{
				Parameters: []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
					{Value: v3, ParameterKey: "out3"},
				},
			},
		},
	})
	assert.NoError(t, err)

	// Update all three tasks in bulk
	updatedV1, _ := structpb.NewValue("updated1")
	updatedV2, _ := structpb.NewValue("updated2")
	updatedV3, _ := structpb.NewValue("updated3")

	bulkReq := &apiv2beta1.UpdateTasksBulkRequest{
		Tasks: map[string]*apiv2beta1.PipelineTaskDetail{
			task1.GetTaskId(): {
				TaskId: task1.GetTaskId(),
				RunId:  runID,
				Name:   "task1",
				State:  apiv2beta1.PipelineTaskDetail_SUCCEEDED,
				Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{
					Parameters: []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
						{Value: updatedV1, ParameterKey: "out1"},
					},
				},
			},
			task2.GetTaskId(): {
				TaskId: task2.GetTaskId(),
				RunId:  runID,
				Name:   "task2",
				State:  apiv2beta1.PipelineTaskDetail_FAILED,
				Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{
					Parameters: []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
						{Value: updatedV2, ParameterKey: "out2"},
					},
				},
			},
			task3.GetTaskId(): {
				TaskId: task3.GetTaskId(),
				RunId:  runID,
				Name:   "task3",
				State:  apiv2beta1.PipelineTaskDetail_SKIPPED,
				Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{
					Parameters: []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
						{Value: updatedV3, ParameterKey: "out3"},
					},
				},
			},
		},
	}

	resp, err := runSrv.UpdateTasksBulk(context.Background(), bulkReq)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 3, len(resp.GetTasks()))

	// Verify each task was updated correctly
	updatedTask1 := resp.GetTasks()[task1.GetTaskId()]
	assert.NotNil(t, updatedTask1)
	assert.Equal(t, apiv2beta1.PipelineTaskDetail_SUCCEEDED, updatedTask1.GetState())
	assert.Equal(t, "updated1", updatedTask1.GetOutputs().GetParameters()[0].GetValue().AsInterface())

	updatedTask2 := resp.GetTasks()[task2.GetTaskId()]
	assert.NotNil(t, updatedTask2)
	assert.Equal(t, apiv2beta1.PipelineTaskDetail_FAILED, updatedTask2.GetState())
	assert.Equal(t, "updated2", updatedTask2.GetOutputs().GetParameters()[0].GetValue().AsInterface())

	updatedTask3 := resp.GetTasks()[task3.GetTaskId()]
	assert.NotNil(t, updatedTask3)
	assert.Equal(t, apiv2beta1.PipelineTaskDetail_SKIPPED, updatedTask3.GetState())
	assert.Equal(t, "updated3", updatedTask3.GetOutputs().GetParameters()[0].GetValue().AsInterface())

	// Verify updates persisted by fetching individually
	fetched1, err := runSrv.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: task1.GetTaskId()})
	assert.NoError(t, err)
	assert.Equal(t, apiv2beta1.PipelineTaskDetail_SUCCEEDED, fetched1.GetState())

	fetched2, err := runSrv.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: task2.GetTaskId()})
	assert.NoError(t, err)
	assert.Equal(t, apiv2beta1.PipelineTaskDetail_FAILED, fetched2.GetState())

	fetched3, err := runSrv.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: task3.GetTaskId()})
	assert.NoError(t, err)
	assert.Equal(t, apiv2beta1.PipelineTaskDetail_SKIPPED, fetched3.GetState())
}

func TestUpdateTasksBulk_EmptyRequest(t *testing.T) {
	clients, manager, _ := seedOneRun(t)
	defer clients.Close()

	runSrv := createRunServer(manager)

	// Test with nil request
	_, err := runSrv.UpdateTasksBulk(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must contain at least one task")

	// Test with empty tasks map
	_, err = runSrv.UpdateTasksBulk(context.Background(), &apiv2beta1.UpdateTasksBulkRequest{
		Tasks: map[string]*apiv2beta1.PipelineTaskDetail{},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must contain at least one task")
}

func TestUpdateTasksBulk_ValidationErrors(t *testing.T) {
	clients, manager, runID := seedOneRun(t)
	defer clients.Close()

	runSrv := createRunServer(manager)

	// Create a task first
	task, err := runSrv.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		Task: &apiv2beta1.PipelineTaskDetail{
			RunId: runID,
			Name:  "test-task",
			State: apiv2beta1.PipelineTaskDetail_RUNNING,
		},
	})
	assert.NoError(t, err)

	// Test with mismatched task IDs
	_, err = runSrv.UpdateTasksBulk(context.Background(), &apiv2beta1.UpdateTasksBulkRequest{
		Tasks: map[string]*apiv2beta1.PipelineTaskDetail{
			task.GetTaskId(): {
				TaskId: "different-id", // Mismatch!
				RunId:  runID,
				State:  apiv2beta1.PipelineTaskDetail_SUCCEEDED,
			},
		},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not match")

	// Test with artifact updates (should fail)
	_, err = runSrv.UpdateTasksBulk(context.Background(), &apiv2beta1.UpdateTasksBulkRequest{
		Tasks: map[string]*apiv2beta1.PipelineTaskDetail{
			task.GetTaskId(): {
				TaskId: task.GetTaskId(),
				RunId:  runID,
				State:  apiv2beta1.PipelineTaskDetail_SUCCEEDED,
				Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{
					Artifacts: []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact{
						{ArtifactKey: "should-fail"},
					},
				},
			},
		},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot update task output artifacts")

	// Test with non-existent task
	_, err = runSrv.UpdateTasksBulk(context.Background(), &apiv2beta1.UpdateTasksBulkRequest{
		Tasks: map[string]*apiv2beta1.PipelineTaskDetail{
			"non-existent-task-id": {
				TaskId: "non-existent-task-id",
				RunId:  runID,
				State:  apiv2beta1.PipelineTaskDetail_SUCCEEDED,
			},
		},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to get existing task")
}
