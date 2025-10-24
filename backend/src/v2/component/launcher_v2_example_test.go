package component

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	clientmanager "github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/client-go/kubernetes/fake"
)

// Example_launcherV2WithMocks demonstrates how to test LauncherV2.Execute with all dependencies mocked.
// This example shows the complete pattern for component-level testing.
func TestExample_launcherV2WithMocks(t *testing.T) {
	// Step 1: Create mock KFP API
	mockAPI := kfpapi.NewMockAPI()

	// Step 2: Create test run and task
	runID := "test-run-123"
	taskID := "test-task-456"

	run := &apiv2beta1.Run{
		RunId:       runID,
		DisplayName: "test-run",
		State:       apiv2beta1.RuntimeState_RUNNING,
		PipelineSource: &apiv2beta1.Run_PipelineSpec{
			PipelineSpec: &structpb.Struct{},
		},
		Tasks: []*apiv2beta1.PipelineTaskDetail{},
	}
	mockAPI.AddRun(run)

	task := &apiv2beta1.PipelineTaskDetail{
		TaskId:  taskID,
		RunId:   runID,
		Name:    "test-task",
		Status:  apiv2beta1.PipelineTaskDetail_RUNNING,
		Type:    apiv2beta1.PipelineTaskDetail_RUNTIME,
		Inputs:  &apiv2beta1.PipelineTaskDetail_InputOutputs{},
		Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{},
	}

	// Step 3: Create executor input with inputs and outputs
	executorInput := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			ParameterValues: map[string]*structpb.Value{
				"input_param": structpb.NewStringValue("test_value"),
			},
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"input_data": {
					Artifacts: []*pipelinespec.RuntimeArtifact{
						{
							Name: "dataset",
							Uri:  "s3://bucket/input/data.csv",
							Type: &pipelinespec.ArtifactTypeSchema{
								Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{
									SchemaTitle: "system.Dataset",
								},
							},
						},
					},
				},
			},
		},
		Outputs: &pipelinespec.ExecutorInput_Outputs{
			Parameters: map[string]*pipelinespec.ExecutorInput_OutputParameter{
				"output_metric": {
					OutputFile: "/tmp/outputs/output_metric",
				},
			},
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"model": {
					Artifacts: []*pipelinespec.RuntimeArtifact{
						{
							Name: "trained-model",
							Uri:  "s3://bucket/output/model.pkl",
							Type: &pipelinespec.ArtifactTypeSchema{
								Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{
									SchemaTitle: "system.Model",
								},
							},
						},
					},
				},
			},
			OutputFile: "/tmp/kfp_outputs/output_metadata.json",
		},
	}

	executorInputJSON, _ := protojson.Marshal(executorInput)

	// Step 4: Create component spec
	componentSpec := &pipelinespec.ComponentSpec{
		InputDefinitions: &pipelinespec.ComponentInputsSpec{
			Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
				"input_param": {
					ParameterType: pipelinespec.ParameterType_STRING,
				},
			},
		},
		OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
			Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
				"output_metric": {
					ParameterType: pipelinespec.ParameterType_NUMBER_DOUBLE,
				},
			},
		},
	}

	// Step 5: Create task spec
	taskSpec := &pipelinespec.PipelineTaskSpec{
		TaskInfo: &pipelinespec.PipelineTaskInfo{
			Name: "train-model",
		},
	}

	// Step 6: Create launcher options
	opts := &LauncherV2Options{
		Namespace:     "default",
		PodName:       "train-model-pod",
		PodUID:        "pod-uid-123",
		PipelineName:  "training-pipeline",
		PublishLogs:   "false",
		ComponentSpec: componentSpec,
		TaskSpec:      taskSpec,
		ScopePath:     util.ScopePath{},
		Run:           run,
		Task:          task,
	}

	// Step 7: Create launcher with client manager
	clientManager := clientmanager.NewFakeClientManager(fake.NewClientset(), mockAPI)
	launcher, _ := NewLauncherV2(
		string(executorInputJSON),
		[]string{"python", "train.py", "--data", "{{$.inputs.artifacts['input_data'].path}}"},
		opts,
		clientManager,
	)

	// Step 8: Setup mocks for dependencies
	mockFS := NewMockFileSystem()
	mockCmd := NewMockCommandExecutor()
	mockObjStore := NewMockObjectStoreClient()

	// Configure file system with output data
	mockFS.SetFileContent("/tmp/outputs/output_metric", []byte("0.95"))
	mockFS.SetFileContent("/tmp/kfp_outputs/output_metadata.json", []byte("{}"))

	// Configure object store with input data
	mockObjStore.SetArtifact("s3://bucket/input/data.csv", []byte("col1,col2\n1,2\n"))

	// Configure command executor to succeed
	mockCmd.RunError = nil

	// Step 9: Inject mocks into launcher
	launcher.WithFileSystem(mockFS).
		WithCommandExecutor(mockCmd).
		WithObjectStore(mockObjStore)

	// Step 10: Execute the launcher's internal execute method
	ctx := context.Background()
	executorOutput, err := launcher.execute(ctx, "python", []string{"train.py"})
	require.NotNil(t, executorOutput)
	if err != nil {
		panic(err)
	}

	// Output: Test passed - launcher executed successfully with mocked dependencies
	println("Test passed - launcher executed successfully with mocked dependencies")
}

// TestLauncherV2_ArtifactHandling demonstrates testing artifact download and upload
func TestLauncherV2_ArtifactHandling(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockObjStore := NewMockObjectStoreClient()

	// Simulate pre-existing input artifact
	mockObjStore.SetArtifact("s3://bucket/input/dataset.csv", []byte("training,data"))

	// Test download
	err := mockObjStore.DownloadArtifact(ctx, "s3://bucket/input/dataset.csv", "/local/dataset.csv", "input_data")
	require.NoError(t, err)

	// Verify download was called with correct parameters
	assert.Len(t, mockObjStore.DownloadCalls, 1)
	assert.Equal(t, "input_data", mockObjStore.DownloadCalls[0].ArtifactKey)
	assert.Equal(t, "s3://bucket/input/dataset.csv", mockObjStore.DownloadCalls[0].RemoteURI)
	assert.Equal(t, "/local/dataset.csv", mockObjStore.DownloadCalls[0].LocalPath)

	// Test upload
	err = mockObjStore.UploadArtifact(ctx, "/local/model.pkl", "s3://bucket/output/model.pkl", "model_output")
	require.NoError(t, err)

	// Verify upload was called
	assert.Len(t, mockObjStore.UploadCalls, 1)
	assert.Equal(t, "model_output", mockObjStore.UploadCalls[0].ArtifactKey)

	// Verify artifact can be queried
	modelUploads := mockObjStore.GetUploadCallsForKey("model_output")
	assert.Len(t, modelUploads, 1)
	assert.Equal(t, "s3://bucket/output/model.pkl", modelUploads[0].RemoteURI)
}

// TestLauncherV2_CommandExecution demonstrates testing command execution
func TestLauncherV2_CommandExecution(t *testing.T) {
	mockCmd := NewMockCommandExecutor()

	// Setup custom behavior to write to stdout
	mockCmd.RunFunc = func(ctx context.Context, cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
		// Simulate successful execution
		stdout.Write([]byte("Training completed successfully\n"))
		stdout.Write([]byte("Accuracy: 0.95\n"))
		return nil
	}

	// Execute command
	ctx := context.Background()
	var stdout, stderr bytes.Buffer
	err := mockCmd.Run(ctx, "python", []string{"train.py"}, nil, &stdout, &stderr)

	// Verify
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Training completed successfully")
	assert.Contains(t, stdout.String(), "Accuracy: 0.95")

	// Verify command was called correctly
	assert.Equal(t, 1, mockCmd.CallCount())
	assert.Equal(t, "python", mockCmd.RunCalls[0].Cmd)
	assert.Equal(t, []string{"train.py"}, mockCmd.RunCalls[0].Args)
}

// TestLauncherV2_FileSystemOperations demonstrates testing file system operations
func TestLauncherV2_FileSystemOperations(t *testing.T) {
	mockFS := NewMockFileSystem()

	// Test directory creation
	err := mockFS.MkdirAll("/tmp/outputs", 0755)
	require.NoError(t, err)

	// Test file writing
	err = mockFS.WriteFile("/tmp/outputs/metrics.json", []byte(`{"accuracy": 0.95}`), 0644)
	require.NoError(t, err)

	// Test file reading
	content, err := mockFS.ReadFile("/tmp/outputs/metrics.json")
	require.NoError(t, err)
	assert.Equal(t, `{"accuracy": 0.95}`, string(content))

	// Verify all operations were tracked
	assert.Len(t, mockFS.MkdirAllCalls, 1)
	assert.Equal(t, "/tmp/outputs", mockFS.MkdirAllCalls[0].Path)

	assert.Len(t, mockFS.WriteFileCalls, 1)
	assert.Equal(t, "/tmp/outputs/metrics.json", mockFS.WriteFileCalls[0].Name)

	assert.Len(t, mockFS.ReadFileCalls, 1)
	assert.Equal(t, "/tmp/outputs/metrics.json", mockFS.ReadFileCalls[0])
}

// TestLauncherV2_TaskStatusUpdates demonstrates testing KFP API task updates
func TestLauncherV2_TaskStatusUpdates(t *testing.T) {
	// Create mock API
	mockAPI := kfpapi.NewMockAPI()

	// Create test run
	run := &apiv2beta1.Run{
		RunId:       "run-123",
		DisplayName: "test-run",
		State:       apiv2beta1.RuntimeState_RUNNING,
		PipelineSource: &apiv2beta1.Run_PipelineSpec{
			PipelineSpec: &structpb.Struct{},
		},
	}
	mockAPI.AddRun(run)

	// Create test task
	task := &apiv2beta1.PipelineTaskDetail{
		TaskId: "task-456",
		RunId:  "run-123",
		Name:   "test-task",
		Status: apiv2beta1.PipelineTaskDetail_RUNNING,
	}
	_, err := mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{Task: task})
	require.NoError(t, err)

	// Update task status
	task.Status = apiv2beta1.PipelineTaskDetail_SUCCEEDED
	_, err = mockAPI.UpdateTask(context.Background(), &apiv2beta1.UpdateTaskRequest{
		TaskId: "task-456",
		Task:   task,
	})
	require.NoError(t, err)

	// Verify task was updated
	updatedTask, err := mockAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: "task-456"})
	require.NoError(t, err)
	assert.Equal(t, apiv2beta1.PipelineTaskDetail_SUCCEEDED, updatedTask.Status)
}
