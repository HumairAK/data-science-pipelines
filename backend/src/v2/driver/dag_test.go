package driver

import (
	"context"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRootDagComponentInputs(t *testing.T) {
	runtimeConfig := &pipelinespec.PipelineJob_RuntimeConfig{
		ParameterValues: map[string]*structpb.Value{
			"string_input": structpb.NewStringValue("test-input1"),
			"number_input": structpb.NewNumberValue(42.5),
			"bool_input":   structpb.NewBoolValue(true),
			"null_input":   structpb.NewNullValue(),
			"list_input": structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{
				structpb.NewStringValue("value1"),
				structpb.NewNumberValue(42),
				structpb.NewBoolValue(true),
			}}),
			"map_input": structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"key1": structpb.NewStringValue("value1"),
					"key2": structpb.NewNumberValue(42),
					"key3": structpb.NewListValue(&structpb.ListValue{
						Values: []*structpb.Value{
							structpb.NewStringValue("nested1"),
							structpb.NewStringValue("nested2"),
						},
					}),
				},
			}),
		},
	}

	tc := NewTestContextWithRootExecuted(t, runtimeConfig, "test_data/taskOutputArtifact_test.py.yaml")
	task := tc.RootTask
	require.NotNil(t, task.Inputs)
	require.NotEmpty(t, task.Inputs.Parameters)

	// Verify parameter values
	paramMap := make(map[string]*structpb.Value)
	for _, param := range task.Inputs.Parameters {
		paramMap[param.GetParameterKey()] = param.Value
	}

	assert.Equal(t, "test-input1", paramMap["string_input"].GetStringValue())
	assert.Equal(t, 42.5, paramMap["number_input"].GetNumberValue())
	assert.Equal(t, true, paramMap["bool_input"].GetBoolValue())
	assert.NotNil(t, paramMap["null_input"].GetNullValue())
	assert.Len(t, paramMap["list_input"].GetListValue().Values, 3)
	assert.NotNil(t, paramMap["map_input"].GetStructValue())
	assert.Len(t, paramMap["map_input"].GetStructValue().Fields, 3)
}

func TestLoopArtifactPassing(t *testing.T) {
	tc := NewTestContextWithRootExecuted(
		t,
		&pipelinespec.PipelineJob_RuntimeConfig{},
		"test_data/loop_collected_raw_Iterator.py.yaml",
	)
	parentTask := tc.RootTask

	// Run Dag on the First Task
	secondaryPipelineExecution, secondaryPipelineTask := tc.RunDag("secondary-pipeline", parentTask)
	require.Nil(t, secondaryPipelineExecution.ExecutorInput.Outputs)
	require.Equal(t, apiv2beta1.PipelineTaskDetail_RUNNING, secondaryPipelineTask.Status)

	// Refresh Parent Task - The parent task should be the secondary pipeline task for "create-dataset"
	parentTask = secondaryPipelineTask

	// Now we'll run the subtasks in the secondary pipeline, one of which is a loop of 3 iterations

	// Run the Downstream Task that will use the output artifact
	createDataSetExecution, _ := tc.RunContainer("create-dataset", parentTask, nil, true)
	require.Nil(t, createDataSetExecution.ExecutorInput.Outputs)

	// Mock a Launcher run by updating the task with output data
	createDataSetOutputArtifactID := tc.MockLauncherOutputArtifactCreate(
		createDataSetExecution.TaskID,
		"output_dataset",
		apiv2beta1.Artifact_Dataset,
		apiv2beta1.IOType_OUTPUT,
		"create-dataset",
		nil,
	)

	// Run the Loop Task - note that parentTask for for-loop-2 remains as secondary-pipeline
	loopExecution, loopTask := tc.RunDag("for-loop-2", parentTask)
	require.Nil(t, secondaryPipelineExecution.ExecutorInput.Outputs)
	require.NotZero(t, len(loopTask.Inputs.Parameters))
	// Expect loop task to have resolved its input parameter
	require.Equal(t, "pipelinechannel--loop-item-param-1", loopTask.Inputs.Parameters[0].ParameterKey)
	// Expect the artifact output of create-dataset as input to for-loop-2
	require.Equal(t, len(loopTask.Inputs.Artifacts), 1)

	// The parent task should be "for-loop-2" for the iterations at first depth
	parentTask = loopTask

	// Perform the iteration calls, mock any launcher calls
	for index, paramID := range []string{"1", "2", "3"} {

		// Run the "process-dataset" Container Task with iteration index
		processExecution, _ := tc.RunContainer("process-dataset", parentTask, util.Int64Pointer(int64(index)), true)
		require.Nil(t, processExecution.ExecutorInput.Outputs)
		require.NotNil(t, processExecution.ExecutorInput.Inputs.Artifacts["input_dataset"])
		require.Equal(t, 1, len(processExecution.ExecutorInput.Inputs.Artifacts["input_dataset"].GetArtifacts()))
		require.Equal(t, processExecution.ExecutorInput.Inputs.Artifacts["input_dataset"].GetArtifacts()[0].ArtifactId, createDataSetOutputArtifactID)
		require.NotNil(t, processExecution.ExecutorInput.Inputs.ParameterValues["model_id_in"])
		require.Equal(t, processExecution.ExecutorInput.Inputs.ParameterValues["model_id_in"].GetStringValue(), paramID)

		// Mock the Launcher run
		processDataSetArtifactID := tc.MockLauncherOutputArtifactCreate(
			processExecution.TaskID,
			"output_artifact",
			apiv2beta1.Artifact_Artifact,
			apiv2beta1.IOType_OUTPUT,
			"process-dataset",
			util.Int64Pointer(int64(index)),
		)

		// Mock: Also expect Launcher->API Server to upload the output artifact to the for-loop-2 task's outputs (by first checking if this artifact is an output artifact)
		//   comp-for-loop-2:
		//    dag:
		//      outputs:
		//        artifacts:
		//          pipelinechannel--process-dataset-output_artifact:
		//            artifactSelectors:
		//            - outputArtifactKey: output_artifact
		//              producerSubtask: process-dataset
		tc.MockLauncherArtifactTaskCreate(
			"process-dataset",
			loopExecution.TaskID,
			"pipelinechannel--process-dataset-output_artifact",
			processDataSetArtifactID,
			util.Int64Pointer(int64(index)),
			apiv2beta1.IOType_ITERATOR_OUTPUT,
		)
		loopTask, err := tc.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: loopExecution.TaskID})
		require.NoError(t, err)
		require.NotNil(t, loopTask.Outputs)
		require.Equal(t, len(loopTask.Outputs.Artifacts), index+1)

		// Mock: Launcher->API Server should also traverse the dag up, to log any output artifacts that are being sourced from the current loop task
		// In this case, secondary-pipeline requires dsl.Collected() sourced from "process-dataset" outputs that are output from for-loop-2
		//   comp-secondary-pipeline:
		//    dag:
		//      outputs:
		//        artifacts:
		//          Output:
		//            artifactSelectors:
		//            - outputArtifactKey: pipelinechannel--process-dataset-output_artifact
		//              producerSubtask: for-loop-2
		tc.MockLauncherArtifactTaskCreate(
			"process-dataset",
			secondaryPipelineExecution.TaskID,
			"Output",
			processDataSetArtifactID,
			util.Int64Pointer(int64(index)),
			apiv2beta1.IOType_ITERATOR_OUTPUT,
		)
		secondaryPipelineTask, err = tc.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: secondaryPipelineExecution.TaskID})
		require.NoError(t, err)
		require.NotNil(t, secondaryPipelineTask.Outputs)
		require.Equal(t, len(secondaryPipelineTask.Outputs.Artifacts), index+1)

		// Run next iteration component
		analyzeExecution, _ := tc.RunContainer("analyze-artifact", parentTask, util.Int64Pointer(int64(index)), true)
		require.Nil(t, createDataSetExecution.ExecutorInput.Outputs)
		require.Nil(t, analyzeExecution.ExecutorInput.Outputs)
		require.NotNil(t, analyzeExecution.ExecutorInput.Inputs.Artifacts["analyze_artifact_input"])
		require.Equal(t, 1, len(analyzeExecution.ExecutorInput.Inputs.Artifacts["analyze_artifact_input"].GetArtifacts()))
		require.Equal(t, analyzeExecution.ExecutorInput.Inputs.Artifacts["analyze_artifact_input"].GetArtifacts()[0].ArtifactId, processDataSetArtifactID)

		// Mock the Launcher run
		_ = tc.MockLauncherOutputArtifactCreate(
			processExecution.TaskID,
			"analyze_output_artifact",
			apiv2beta1.Artifact_Artifact,
			apiv2beta1.IOType_OUTPUT,
			"analyze-artifact",
			util.Int64Pointer(int64(index)),
		)
	}

	tasks, err := tc.DriverAPI.ListTasks(context.Background(), &apiv2beta1.ListTasksRequest{
		ParentFilter: &apiv2beta1.ListTasksRequest_ParentId{ParentId: loopExecution.TaskID},
	})
	require.NoError(t, err)
	require.NotNil(t, tasks)
	// Expect 3 tasks for analyze-artifact + 3 tasks for process-dataset
	require.Equal(t, 6, len(tasks.Tasks))

	// Expect the 3 artifacts from process-task to have been collected by the for-loop-2 task
	forLoopTask, err := tc.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: loopExecution.TaskID})
	require.NoError(t, err)
	require.Equal(t, 3, len(forLoopTask.Outputs.Artifacts))

	// Run "analyze_artifact_list" in "secondary_pipeline"

	// Move up a parent
	parentTask = secondaryPipelineTask
	tc.ExitDag()

	analyzeArtifactListExecution, _ := tc.RunContainer("analyze-artifact-list", parentTask, nil, true)
	require.Nil(t, analyzeArtifactListExecution.ExecutorInput.Outputs)
	require.NotNil(t, analyzeArtifactListExecution.ExecutorInput.Inputs.Artifacts["artifact_list_input"])
	require.Equal(t, 3, len(analyzeArtifactListExecution.ExecutorInput.Inputs.Artifacts["artifact_list_input"].GetArtifacts()))

	// Primary Pipeline tests

	// Expect the 3 artifacts from process-task to have been collected by the secondary-pipeline task
	secondaryPipelineTask, err = tc.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: secondaryPipelineExecution.TaskID})
	require.NoError(t, err)
	require.Equal(t, 3, len(secondaryPipelineTask.Outputs.Artifacts))

	// Move up a parent
	parentTask = tc.RootTask
	tc.ExitDag()

	// Not to be confused with the "analyze-artifact-list" task in secondary pipeline,
	// this is the "analyze-artifact-list" task in the primary pipeline
	analyzeArtifactListOuterExecution, _ := tc.RunContainer("analyze-artifact-list", parentTask, nil, true)
	require.Nil(t, analyzeArtifactListExecution.ExecutorInput.Outputs)
	require.Nil(t, analyzeArtifactListOuterExecution.ExecutorInput.Outputs)
	require.NotNil(t, analyzeArtifactListOuterExecution.ExecutorInput.Inputs.Artifacts["artifact_list_input"])
	require.Equal(t, 3, len(analyzeArtifactListOuterExecution.ExecutorInput.Inputs.Artifacts["artifact_list_input"].GetArtifacts()))

	// Refresh Run so it has the new tasks
	tc.RefreshRun()

	// primary_pipeline()		 x 1  (root)
	// secondary_pipeline()      x 1  (dag)
	//   create_dataset()        x 1  (runtime)
	//   for_loop_1()            x 1  (loop)
	//     process_dataset()     x 3  (runtime)
	//	   analyze_artifact()    x 3  (runtime)
	//   analyze_artifact_list() x 1  (runtime)
	// analyze_artifact_list()   x 1  (runtime)
	require.Equal(t, 12, len(tc.Run.Tasks))
}

// TestParameterInputIterator will test parameter Input Iterator
// and parameter collection from output of a task in a loop
func TestParameterInputIterator(t *testing.T) {
	tc := NewTestContextWithRootExecuted(
		t,
		&pipelinespec.PipelineJob_RuntimeConfig{},
		"test_data/loop_collected_InputParameter_Iterator.py.yaml",
	)
	// Execute full pipeline
	parentTask := tc.RootTask
	_, secondaryPipelineTask := tc.RunDag("secondary-pipeline", parentTask)
	parentTask = secondaryPipelineTask
	_, splitIDsTask := tc.RunContainer("split-ids", parentTask, nil, true)

	tc.MockLauncherOutputParameterCreate(
		splitIDsTask.GetTaskId(),
		"Output",
		&structpb.Value{
			Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{
				Values: []*structpb.Value{
					structpb.NewStringValue("1"),
					structpb.NewStringValue("2"),
					structpb.NewStringValue("3"),
				},
			},
			},
		},
		apiv2beta1.IOType_OUTPUT,
		"split-ids",
		nil,
	)

	_, loopTask := tc.RunDag("for-loop-1", parentTask)
	parentTask = loopTask

	for index, _ := range []string{"1", "2", "3"} {
		index64 := util.Int64Pointer(int64(index))
		_, createFileTask := tc.RunContainer(
			"create-file",
			parentTask,
			index64,
			true,
		)

		tc.MockLauncherOutputArtifactCreate(
			createFileTask.GetTaskId(),
			"file",
			apiv2beta1.Artifact_Artifact,
			apiv2beta1.IOType_OUTPUT,
			"create-file",
			index64,
		)

		// Run next task
		_, readSingleFileTask := tc.RunContainer(
			"read-single-file",
			parentTask,
			index64,
			true,
		)
		mockSingleFileTaskOutputParameterValue := &structpb.Value{
			Kind: &structpb.Value_StringValue{
				StringValue: fmt.Sprintf("file-%d", index),
			},
		}
		tc.MockLauncherOutputParameterCreate(
			readSingleFileTask.GetTaskId(),
			"Output",
			mockSingleFileTaskOutputParameterValue,
			apiv2beta1.IOType_ITERATOR_OUTPUT,
			"read-single-file",
			index64,
		)

		// Parameter should be also sent upstream for collection
		tc.MockLauncherOutputParameterCreate(
			loopTask.GetTaskId(),
			"pipelinechannel--read-single-file-Output",
			mockSingleFileTaskOutputParameterValue,
			apiv2beta1.IOType_ITERATOR_OUTPUT,
			"read-single-file",
			index64,
		)
		tc.MockLauncherOutputParameterCreate(
			secondaryPipelineTask.GetTaskId(),
			"Output",
			mockSingleFileTaskOutputParameterValue,
			apiv2beta1.IOType_ITERATOR_OUTPUT,
			"read-single-file",
			index64,
		)

	}

	tc.ExitDag()
	parentTask = secondaryPipelineTask

	_, readValuesTask := tc.RunContainer("read-values", parentTask, nil, true)
	tc.MockLauncherOutputParameterCreate(
		readValuesTask.GetTaskId(),
		"Output",
		&structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "files read"}},
		apiv2beta1.IOType_OUTPUT,
		"read-values",
		nil,
	)

	tc.ExitDag()
	parentTask = tc.RootTask

	_, readValuesTask2 := tc.RunContainer("read-values", parentTask, nil, true)
	tc.MockLauncherOutputParameterCreate(
		readValuesTask2.GetTaskId(),
		"Output",
		&structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "files read"}},
		apiv2beta1.IOType_OUTPUT,
		"read-values",
		nil,
	)

	task, err := tc.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: secondaryPipelineTask.GetTaskId()})
	require.NoError(t, err)
	require.NotNil(t, task.Outputs)
	require.Equal(t, 3, len(task.Outputs.Parameters))
	var collectOutputs []string
	for _, params := range task.Outputs.Parameters {
		collectOutputs = append(collectOutputs, params.GetValue().GetStringValue())
		require.Equal(t, apiv2beta1.IOType_ITERATOR_OUTPUT, params.GetType())
	}
	require.Equal(t, []string{"file-0", "file-1", "file-2"}, collectOutputs)
}

func TestNestedDag(t *testing.T) {
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/nested_naming_conflicts.py.yaml")
	parentTask := tc.RootTask

	_, aTask := tc.RunContainer("a", parentTask, nil, true)
	tc.MockLauncherOutputArtifactCreate(
		aTask.GetTaskId(),
		"output_dataset",
		apiv2beta1.Artifact_Dataset,
		apiv2beta1.IOType_OUTPUT,
		"a",
		nil)

	_, pipelineBTask := tc.RunDag("pipeline-b", parentTask)
	parentTask = pipelineBTask

	_, nestedATask := tc.RunContainer("a", parentTask, nil, true)
	tc.MockLauncherOutputArtifactCreate(
		nestedATask.GetTaskId(),
		"output_dataset",
		apiv2beta1.Artifact_Dataset,
		apiv2beta1.IOType_OUTPUT,
		"a",
		nil)

	_, nestedBTask := tc.RunContainer("b", parentTask, nil, true)
	tc.MockLauncherOutputArtifactCreate(
		nestedBTask.GetTaskId(),
		"output_artifact_b",
		apiv2beta1.Artifact_Artifact,
		apiv2beta1.IOType_OUTPUT,
		"b",
		nil)

	_, pipelineCTask := tc.RunDag("pipeline-c", parentTask)
	parentTask = pipelineCTask

	_, nestedNestedATask := tc.RunContainer("a", parentTask, nil, true)
	tc.MockLauncherOutputArtifactCreate(
		nestedNestedATask.GetTaskId(),
		"output_dataset",
		apiv2beta1.Artifact_Dataset,
		apiv2beta1.IOType_OUTPUT,
		"a",
		nil)

	_, nestedNestedBTask := tc.RunContainer("b", parentTask, nil, true)
	tc.MockLauncherOutputArtifactCreate(
		nestedNestedBTask.GetTaskId(),
		"output_artifact_b",
		apiv2beta1.Artifact_Artifact,
		apiv2beta1.IOType_OUTPUT,
		"b",
		nil)

	_, cTask := tc.RunContainer("c", parentTask, nil, true)
	cTaskArtifactID := tc.MockLauncherOutputArtifactCreate(
		cTask.GetTaskId(),
		"output_artifact_c",
		apiv2beta1.Artifact_Artifact,
		apiv2beta1.IOType_OUTPUT,
		"c",
		nil)

	// Dag output for pipeline_b
	tc.MockLauncherArtifactTaskCreate(
		cTask.GetName(),
		pipelineBTask.GetTaskId(),
		"Output",
		cTaskArtifactID,
		nil,
		apiv2beta1.IOType_OUTPUT,
	)

	tc.ExitDag()
	parentTask = pipelineBTask

	tc.ExitDag()
	parentTask = tc.RootTask

	_, _ = tc.RunContainer("verify", parentTask, nil, true)

	var err error

	// Confirm that the artifact passed to "verify" task came from task_c
	pipelineBTask, err = tc.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: pipelineBTask.GetTaskId()})
	require.NoError(t, err)
	require.NotNil(t, pipelineBTask.Outputs)
	require.Equal(t, 1, len(pipelineBTask.Outputs.Artifacts))
	require.Equal(t, cTask.GetTaskId(), pipelineBTask.Outputs.Artifacts[0].GetArtifacts()[0].GetMetadata()["task_id"].GetStringValue())

	// Confirm that the artifact passed to cTask came from the nestedNestedBtask
	// I.e the b() task that ran in pipeline-c and not in pipeline-b
	cTask, err = tc.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: cTask.GetTaskId()})
	require.NoError(t, err)
	require.NotNil(t, cTask.Outputs)
	require.Equal(t, 1, len(cTask.Outputs.Artifacts))
	require.Equal(t, nestedNestedBTask.GetTaskId(), cTask.Inputs.Artifacts[0].GetArtifacts()[0].GetMetadata()["task_id"].GetStringValue())
}

func TestParameterTaskOutput(t *testing.T) {
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/taskOutputParameter_test.py.yaml")
	parentTask := tc.RootTask

	// Run Dag on the First Task
	cdExecution, _ := tc.RunContainer("create-dataset", parentTask, nil, true)
	tc.MockLauncherOutputParameterCreate(
		cdExecution.TaskID,
		"output_parameter_path",
		&structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: 10.0}},
		apiv2beta1.IOType_OUTPUT,
		"create-dataset",
		nil,
	)
	pdExecution, _ := tc.RunContainer("process-dataset", parentTask, nil, true)
	tc.MockLauncherOutputParameterCreate(
		pdExecution.TaskID,
		"output_int",
		&structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "output_int_value"}},
		apiv2beta1.IOType_OUTPUT,
		"process-dataset",
		nil,
	)
	analyzeArtifactExecution, _ := tc.RunContainer("analyze-artifact", parentTask, nil, true)
	tc.MockLauncherOutputParameterCreate(
		analyzeArtifactExecution.TaskID,
		"output_opinion",
		&structpb.Value{Kind: &structpb.Value_BoolValue{BoolValue: true}},
		apiv2beta1.IOType_OUTPUT,
		"analyze-artifact",
		nil,
	)
}

func TestOneOf(t *testing.T) {
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/oneof_simple.yaml")
	parentTask := tc.RootTask
	require.NotNil(t, parentTask)

	// Run secondary pipeline
	_, secondaryPipelineTask := tc.RunDag("secondary-pipeline", parentTask)
	parentTask = secondaryPipelineTask

	// Run create_dataset()
	_, createDatasetTask := tc.RunContainer("create-dataset", parentTask, nil, true)
	tc.MockLauncherOutputArtifactCreate(
		createDatasetTask.GetTaskId(),
		"output_dataset",
		apiv2beta1.Artifact_Dataset,
		apiv2beta1.IOType_OUTPUT,
		createDatasetTask.GetName(),
		nil,
	)
	tc.MockLauncherOutputParameterCreate(
		createDatasetTask.GetTaskId(),
		"condition_out",
		&structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "second"}},
		apiv2beta1.IOType_OUTPUT,
		createDatasetTask.GetName(),
		nil,
	)

	// Run ConditionBranch
	_, conditionBranch1Task := tc.RunDag("condition-branches-1", parentTask)
	parentTask = conditionBranch1Task

	// Expect this condition to not be met
	condition2Execution, _ := tc.RunDag("condition-2", conditionBranch1Task)
	require.NotNil(t, condition2Execution.Condition)
	require.False(t, *condition2Execution.Condition)

	tc.ExitDag()

	// Expect this condition to not be met
	condition4Execution, _ := tc.RunDag("condition-4", conditionBranch1Task)
	require.NotNil(t, condition4Execution.Condition)
	require.False(t, *condition4Execution.Condition)

	tc.ExitDag()

	// Expect this condition to pass since output of
	// create-dataset == "second"
	condition3Execution, condition3Task := tc.RunDag("condition-3", conditionBranch1Task)
	require.NotNil(t, condition3Execution.Condition)
	require.True(t, *condition3Execution.Condition)

	parentTask = condition3Task
	_, giveAnimal1Task := tc.RunContainer("give-animal-2", parentTask, nil, true)
	tc.MockLauncherOutputArtifactCreate(
		giveAnimal1Task.GetTaskId(),
		"output_animal",
		apiv2beta1.Artifact_Artifact,
		apiv2beta1.IOType_OUTPUT,
		giveAnimal1Task.GetName(),
		nil,
	)
	_, analyzeAnimal1Task := tc.RunContainer("analyze-animal", parentTask, nil, true)
	analyzeAnimal1TaskArtifactID := tc.MockLauncherOutputArtifactCreate(
		analyzeAnimal1Task.GetTaskId(),
		"analysis_output",
		apiv2beta1.Artifact_Artifact,
		apiv2beta1.IOType_OUTPUT,
		analyzeAnimal1Task.GetName(),
		nil,
	)

	// Expect launcher to create artifact task to
	// analyzeAnimal1TaskArtifactID. Launcher should
	// search through the artifactSelectors for the
	// parentTask, findthe matching outputArtifactKey
	// the outputArtifactKey in the selector should match
	// the outputDefinition key of comp-condition-3.
	tc.MockLauncherArtifactTaskCreate(
		analyzeAnimal1Task.GetName(),
		conditionBranch1Task.GetTaskId(),
		"pipelinechannel--condition-branches-1-oneof-2",
		analyzeAnimal1TaskArtifactID,
		nil,
		apiv2beta1.IOType_ONE_OF_OUTPUT,
	)

	// It is also an output of the secondary pipeline
	tc.MockLauncherArtifactTaskCreate(
		analyzeAnimal1Task.GetName(),
		secondaryPipelineTask.GetTaskId(),
		"Output",
		analyzeAnimal1TaskArtifactID,
		nil,
		apiv2beta1.IOType_ONE_OF_OUTPUT,
	)

	tc.ExitDag()
	parentTask = conditionBranch1Task
	tc.ExitDag()
	parentTask = secondaryPipelineTask
	tc.ExitDag()
	parentTask = tc.RootTask

	_, _ = tc.RunContainer("check-animal", parentTask, nil, true)
}

func TestFinalStatus(t *testing.T) {
	tc := NewTestContextWithRootExecuted(
		t,
		&pipelinespec.PipelineJob_RuntimeConfig{},
		"test_data/pipeline_with_input_status_state.yaml",
	)
	parentTask := tc.RootTask
	require.NotNil(t, parentTask)

	_, exitHandler1Task := tc.RunDag("exit-handler-1", parentTask)
	parentTask = exitHandler1Task

	_, _ = tc.RunContainer("some-task", parentTask, nil, true)

	tc.ExitDag()
	parentTask = tc.RootTask

	_, echoStateTask := tc.RunContainer("echo-state", parentTask, nil, true)
	require.Len(t, echoStateTask.Inputs.GetParameters(), 1)
	inputFinalStatusParam := echoStateTask.Inputs.GetParameters()[0]
	require.NotNil(t, inputFinalStatusParam)
	// Mock library doesn't update dag statuses, in production we would expect
	// this to say "SUCCEEDED" once it's done running
	require.Equal(t, "RUNNING", inputFinalStatusParam.GetValue().GetStructValue().Fields["state"].GetStringValue())
	require.Equal(t, "exit-handler-1", inputFinalStatusParam.GetValue().GetStructValue().Fields["pipelineTaskName"].GetStringValue())
}

func TestWithCaching(t *testing.T) {
	tc := NewTestContextWithRootExecuted(
		t,
		&pipelinespec.PipelineJob_RuntimeConfig{},
		"test_data/cache_test.py.yaml",
	)
	parentTask := tc.RootTask
	require.NotNil(t, parentTask)

	_, createDatasetTask := tc.RunContainer("create-dataset", parentTask, nil, true)
	tc.MockLauncherOutputArtifactCreate(
		createDatasetTask.GetTaskId(),
		"output_dataset",
		apiv2beta1.Artifact_Dataset,
		apiv2beta1.IOType_OUTPUT,
		createDatasetTask.GetName(),
		nil)

	processDatasetExecution, processDatasetTask := tc.RunContainer("process-dataset", parentTask, nil, true)
	tc.MockLauncherOutputArtifactCreate(
		processDatasetTask.GetTaskId(),
		"output_artifact",
		apiv2beta1.Artifact_Artifact,
		apiv2beta1.IOType_OUTPUT,
		processDatasetTask.GetName(),
		nil)
	require.NotNil(t, processDatasetExecution.Cached)
	require.False(t, *processDatasetExecution.Cached)
	require.Equal(t, apiv2beta1.PipelineTaskDetail_SUCCEEDED, processDatasetTask.GetStatus())
	require.NotEmpty(t, processDatasetExecution.PodSpecPatch)

	processDatasetExecution, processDatasetTask = tc.RunContainer("process-dataset", parentTask, nil, true)
	require.NotNil(t, processDatasetExecution.Cached)
	require.True(t, *processDatasetExecution.Cached)
	require.Equal(t, apiv2beta1.PipelineTaskDetail_CACHED, processDatasetTask.GetStatus())
	require.Empty(t, processDatasetExecution.PodSpecPatch)
}

func TestOptionalFields(t *testing.T) {
	// The API Server will populate runtime config with
	// the defaults in the root InputDefinition is they are
	// not user overridden. We mock this here.
	runtimeInputs := &pipelinespec.PipelineJob_RuntimeConfig{
		ParameterValues: map[string]*structpb.Value{
			"input_str4": structpb.NewNullValue(),
			"input_str5": structpb.NewStringValue("Some pipeline default"),
			"input_str6": structpb.NewNullValue(),
		},
	}

	tc := NewTestContextWithRootExecuted(
		t, runtimeInputs,
		"test_data/component_with_optional_inputs.yaml",
	)
	parentTask := tc.RootTask
	require.NotNil(t, parentTask)

	execution, task := tc.RunContainer("component-op", parentTask, nil, false)
	require.NotNil(t, task)
	require.NotNil(t, execution)
	task = tc.MockLauncherDefaultInputParametersUpdate(task.TaskId, tc.GetLast().GetComponentSpec())

	params := task.Inputs.GetParameters()
	require.GreaterOrEqual(t, len(params), 0)

	p := tc.fetchParameter("input_str1", params)
	require.NotNil(t, p)

	p = tc.fetchParameter("input_str2", params)
	require.NotNil(t, p)

	p = tc.fetchParameter("input_str3", params)
	require.Nil(t, p)

	p = tc.fetchParameter("input_str4_from_pipeline", params)
	require.NotNil(t, p)

	p = tc.fetchParameter("input_str5_from_pipeline", params)
	require.NotNil(t, p)

	p = tc.fetchParameter("input_str6_from_pipeline", params)
	require.Nil(t, p)

	p = tc.fetchParameter("input_bool1", params)
	require.NotNil(t, p)

	p = tc.fetchParameter("input_bool2", params)
	require.Nil(t, p)

	p = tc.fetchParameter("input_dict", params)
	require.NotNil(t, p)

	p = tc.fetchParameter("input_list", params)
	require.NotNil(t, p)

	p = tc.fetchParameter("input_int", params)
	require.NotNil(t, p)
}

func TestK8SPlatform(t *testing.T) {
	nodeAffinity := structpb.NewStructValue(&structpb.Struct{
		Fields: map[string]*structpb.Value{
			"requiredDuringSchedulingIgnoredDuringExecution": structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"nodeSelectorTerms": structpb.NewListValue(&structpb.ListValue{
						Values: []*structpb.Value{
							structpb.NewStructValue(&structpb.Struct{
								Fields: map[string]*structpb.Value{
									"matchExpressions": structpb.NewListValue(&structpb.ListValue{
										Values: []*structpb.Value{
											structpb.NewStructValue(&structpb.Struct{
												Fields: map[string]*structpb.Value{
													"key":      structpb.NewStringValue("kubernetes.io/os"),
													"operator": structpb.NewStringValue("In"),
													"values": structpb.NewListValue(&structpb.ListValue{
														Values: []*structpb.Value{
															structpb.NewStringValue("linux"),
														},
													}),
												},
											}),
										},
									}),
								},
							}),
						},
					}),
				},
			}),
		},
	})

	// The API Server will populate runtime config with
	// the defaults in the root InputDefinition is they are
	// not user overridden. We mock this here.
	runtimeInputs := &pipelinespec.PipelineJob_RuntimeConfig{
		ParameterValues: map[string]*structpb.Value{
			"configmap_parm":              structpb.NewStringValue("cfg-2"),
			"default_node_affinity_input": nodeAffinity,
			"empty_dir_mnt_path":          structpb.NewStringValue("/empty_dir/path"),
			"field_path":                  structpb.NewStringValue("spec.serviceAccountName"),
			"node_selector_input": structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"kubernetes.io/os": structpb.NewStringValue("linux"),
				},
			}),
			"pull_secret_1":         structpb.NewStringValue("pull-secret-1"),
			"pull_secret_2":         structpb.NewStringValue("pull-secret-2"),
			"pull_secret_3":         structpb.NewStringValue("pull-secret-3"),
			"pvc_name_suffix_input": structpb.NewStringValue("-pvc-1"),
			"secret_param":          structpb.NewStringValue("secret-2"),
			"tolerations_dict_input": structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"effect":   structpb.NewStringValue("NoSchedule"),
					"key":      structpb.NewStringValue("some_foo_key6"),
					"operator": structpb.NewStringValue("Equal"),
					"value":    structpb.NewStringValue("value3"),
				},
			}),
			"tolerations_list_input": structpb.NewListValue(&structpb.ListValue{
				Values: []*structpb.Value{
					structpb.NewStructValue(&structpb.Struct{
						Fields: map[string]*structpb.Value{
							"effect":   structpb.NewStringValue("NoSchedule"),
							"key":      structpb.NewStringValue("some_foo_key4"),
							"operator": structpb.NewStringValue("Equal"),
							"value":    structpb.NewStringValue("value2"),
						},
					}),
					structpb.NewStructValue(&structpb.Struct{
						Fields: map[string]*structpb.Value{
							"effect":   structpb.NewStringValue("NoExecute"),
							"key":      structpb.NewStringValue("some_foo_key5"),
							"operator": structpb.NewStringValue("Exists"),
						},
					}),
				},
			}),
		},
	}

	tc := NewTestContextWithRootExecuted(
		t, runtimeInputs,
		"test_data/k8s_parameters.yaml",
	)
	parentTask := tc.RootTask
	require.NotNil(t, parentTask)

	// Execute all the preliminary tasks that will feed Task Output Parameters to the
	// Assert tasks (and secondary pipeline)
	_, cfgNameGeneratorTask := tc.RunContainer("cfg-name-generator", parentTask, nil, true)
	tc.MockLauncherOutputParameterCreate(
		cfgNameGeneratorTask.TaskId,
		"some_output",
		structpb.NewStringValue("cfg-3"),
		apiv2beta1.IOType_OUTPUT,
		cfgNameGeneratorTask.GetName(),
		nil,
	)
	_, getAccessModeTask := tc.RunContainer("get-access-mode", parentTask, nil, true)
	tc.MockLauncherOutputParameterCreate(
		getAccessModeTask.TaskId,
		"access_mode",
		structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{structpb.NewStringValue("ReadWriteOnce")}}),
		apiv2beta1.IOType_OUTPUT,
		getAccessModeTask.GetName(),
		nil,
	)
	_, getNodeAffinityTask := tc.RunContainer("get-node-affinity", parentTask, nil, true)
	tc.MockLauncherOutputParameterCreate(
		getNodeAffinityTask.TaskId,
		"node_affinity",
		nodeAffinity,
		apiv2beta1.IOType_OUTPUT,
		getNodeAffinityTask.GetName(),
		nil,
	)
	_, secretNameGeneratorTask := tc.RunContainer("secret-name-generator", parentTask, nil, true)
	tc.MockLauncherOutputParameterCreate(
		secretNameGeneratorTask.TaskId,
		"some_output",
		nodeAffinity,
		apiv2beta1.IOType_OUTPUT,
		secretNameGeneratorTask.GetName(),
		nil,
	)

	// Run create-pvc task since it depended on get-access-mode
	// There is no launcher for this task, we expect the output
	// parameter to be created by the driver call

	// Create a mock Kubernetes client for PVC operations
	_, createPvcTask := tc.RunContainer("createpvc", parentTask, nil, true)
	require.NotNil(t, createPvcTask.Outputs)
	require.Len(t, createPvcTask.Outputs.GetParameters(), 1)

	tc.MockLauncherOutputParameterCreate(
		createPvcTask.TaskId,
		"name",
		structpb.NewStringValue("some-name"),
		apiv2beta1.IOType_OUTPUT,
		createPvcTask.GetName(),
		nil,
	)

	_, assertValuesTask := tc.RunContainer("assert-values", parentTask, nil, true)
	require.NotNil(t, assertValuesTask.Outputs)
	require.Len(t, assertValuesTask.Outputs.GetParameters(), 1)

}

func TestArtifactIterator(t *testing.T) {

}

// This test creates a DAG with a single task that uses a component with inputs
// and runtime constants. The test verifies that the inputs are correctly passed
// to the Runtime Task.
func TestContainerComponentInputsAndRuntimeConstants(t *testing.T) {
	// Create a root DAG execution using basic inputs
	runtimeInputs := &pipelinespec.PipelineJob_RuntimeConfig{
		ParameterValues: map[string]*structpb.Value{
			"name_in":      structpb.NewStringValue("some_name"),
			"number_in":    structpb.NewNumberValue(1.0),
			"threshold_in": structpb.NewNumberValue(0.1),
			"active_in":    structpb.NewBoolValue(false),
		},
	}

	tc := NewTestContextWithRootExecuted(t, runtimeInputs, "test_data/componentInput_level_1_test.py.yaml")

	// Run Container on the First Task
	processInputsExecution, processInputsTask := tc.RunContainer("process-inputs", tc.RootTask, nil, true)
	require.Nil(t, processInputsExecution.ExecutorInput.Outputs)

	// Fetch the task created by the Container() call
	params := processInputsTask.Inputs.GetParameters()
	require.Equal(t, apiv2beta1.IOType_COMPONENT_INPUT, tc.fetchParameter("name", params).GetType())
	require.Equal(t, apiv2beta1.IOType_COMPONENT_INPUT, tc.fetchParameter("number", params).GetType())
	require.Equal(t, apiv2beta1.IOType_COMPONENT_INPUT, tc.fetchParameter("active", params).GetType())
	require.Equal(t, apiv2beta1.IOType_COMPONENT_INPUT, tc.fetchParameter("threshold", params).GetType())
	require.Equal(t, apiv2beta1.IOType_RUNTIME_VALUE_INPUT, tc.fetchParameter("a_runtime_string", params).GetType())
	require.Equal(t, apiv2beta1.IOType_RUNTIME_VALUE_INPUT, tc.fetchParameter("a_runtime_number", params).GetType())
	require.Equal(t, apiv2beta1.IOType_RUNTIME_VALUE_INPUT, tc.fetchParameter("a_runtime_bool", params).GetType())

	require.Equal(t, processInputsExecution.TaskID, processInputsTask.TaskId)
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["name"].GetStringValue(), "some_name")
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["number"].GetNumberValue(), 1.0)
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["threshold"].GetNumberValue(), 0.1)
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["active"].GetBoolValue(), false)
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["a_runtime_string"].GetStringValue(), "foo")
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["a_runtime_number"].GetNumberValue(), 10.0)
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["a_runtime_bool"].GetBoolValue(), true)

	// Mock a Launcher run by updating the task with output data
	tc.MockLauncherOutputArtifactCreate(
		processInputsTask.TaskId,
		"output_text",
		apiv2beta1.Artifact_Dataset,
		apiv2beta1.IOType_OUTPUT,
		"process-inputs",
		nil,
	)

	analyzeInputsExecution, _ := tc.RunContainer("analyze-inputs", tc.RootTask, nil, true)
	require.Nil(t, analyzeInputsExecution.ExecutorInput.Outputs)
	require.Nil(t, analyzeInputsExecution.ExecutorInput.Outputs)
	require.Equal(t, 1, len(analyzeInputsExecution.ExecutorInput.Inputs.Artifacts["input_text"].Artifacts))

	// Verify Executor Input has the correct artifact
	artifact := analyzeInputsExecution.ExecutorInput.Inputs.Artifacts["input_text"].Artifacts[0]
	require.NotNil(t, artifact.Metadata)
	require.NotNil(t, artifact.Metadata.GetFields()["display_name"])
	require.Equal(t, "output_text", artifact.Metadata.GetFields()["display_name"].GetStringValue())
	require.Equal(t, "s3://some.location/output_text", artifact.Uri)
	require.Equal(t, apiv2beta1.Artifact_Dataset.String(), artifact.Type.GetSchemaTitle())
	require.Equal(t, "output_text", artifact.Name)
}
