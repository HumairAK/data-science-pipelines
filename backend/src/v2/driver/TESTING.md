# Driver Testing Framework

This document describes the testing framework for the KFP v2 driver, including the TestContext pattern and how to write integration-style tests that exercise both driver and launcher components.

## Table of Contents
- [Overview](#overview)
- [TestContext Pattern](#testcontext-pattern)
- [Driver Testing Methods](#driver-testing-methods)
- [Launcher Testing with RunLauncher](#launcher-testing-with-runlauncher)
- [Mock Infrastructure](#mock-infrastructure)
- [Complete Example](#complete-example)

## Overview

The driver testing framework provides utilities for testing pipeline execution flows that involve:

1. **Driver execution**: Creating tasks and resolving inputs/outputs
2. **Launcher execution**: Simulating component execution with mocked dependencies
3. **API interactions**: Task and artifact creation/updates via MockAPI

The framework uses a hybrid approach:
- **Real driver logic**: Actual driver code runs to create ExecutorInput and tasks
- **Real launcher logic**: Actual launcher code runs to execute components
- **Mocked dependencies**: File system, command execution, and object storage are mocked

## TestContext Pattern

`TestContext` is the central testing utility that manages:

- Pipeline specification and scope path tracking
- Run and task state
- Mock API client for KFP v2beta1 API
- Helper methods for driver and launcher execution

### Creating a TestContext

```go
// Create test context with a root DAG already executed
tc := NewTestContextWithRootExecuted(
    t,
    runtimeConfig,  // Pipeline runtime inputs
    "test_data/pipeline.yaml",  // Path to pipeline spec
)
```

This automatically:
1. Creates a test run
2. Loads the pipeline spec
3. Executes the root DAG driver
4. Sets up the initial scope path

### TestContext Fields

```go
type TestContext struct {
    Run           *apiv2beta1.Run
    ScopePath     util.ScopePath
    T             *testing.T
    PipelineSpec  *pipelinespec.PipelineSpec
    RootTask      *apiv2beta1.PipelineTaskDetail
    PlatformSpec  *pipelinespec.PlatformSpec
    ClientManager clientmanager.ClientManagerInterface
    MockAPI       *kfpapi.MockAPI
}
```

## Driver Testing Methods

### RunContainer

Executes a container driver for a runtime task:

```go
execution, task := tc.RunContainer(
    "task-name",
    parentTask,
    iterationIndex,  // nil for non-loop tasks, or int64 pointer for loop iterations
    autoUpdateScope, // true to auto-pop scope after execution
)
```

Returns:
- `execution *Execution`: Contains ExecutorInput, PodSpecPatch, TaskID, etc.
- `task *apiv2beta1.PipelineTaskDetail`: The created task

**Important**: `execution.ExecutorInput` contains the fully resolved inputs/outputs prepared by the driver.

### RunDag

Executes a DAG driver for a sub-pipeline or loop:

```go
execution, task := tc.RunDag("dag-task-name", parentTask)
```

Returns the same types as RunContainer. Remember to call `tc.ExitDag()` when done with the DAG's children.

### RunRootDag

Executes the root DAG driver (called automatically by NewTestContextWithRootExecuted):

```go
execution, task := tc.RunRootDag(tc, run, runtimeConfig)
```

### Scope Management

```go
// Enter a task scope (automatically done by RunContainer/RunDag)
err := tc.ScopePath.Push("task-name")

// Exit a task scope (for container tasks with autoUpdateScope=false, or after completing DAG children)
tc.ExitDag()

// Access current scope
componentSpec := tc.GetLast().GetComponentSpec()
taskSpec := tc.GetLast().GetTaskSpec()
```

## Launcher Testing with RunLauncher

`RunLauncher` simulates the execution of a launcher container with mocked dependencies. It uses the **real launcher code** but injects mocks for:
- File system operations
- Command execution
- Object store uploads/downloads

### Key Concept: ExecutorInput Reuse

**IMPORTANT**: `RunLauncher` reuses the `ExecutorInput` that was prepared by the driver (via `RunContainer` or `RunDag`). This ensures that the launcher tests the same inputs/outputs that the driver resolved.

```go
// Step 1: Run driver to create task and ExecutorInput
execution, task := tc.RunContainer("create-dataset", parentTask, nil, true)

// Step 2: Run launcher using the driver's ExecutorInput
launcherExec := tc.RunLauncher(execution, map[string][]byte{
    "/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
    "/tmp/outputs/metric": []byte("0.95"),
})
```

### RunLauncher Signature

```go
func (tc *TestContext) RunLauncher(
    execution *Execution,
    outputFiles map[string][]byte,
) *LauncherExecution
```

**Parameters**:
- `execution`: The Execution returned by RunContainer/RunDag (contains the ExecutorInput)
- `outputFiles`: Map of file paths to file contents that the launcher should "see"

**Returns** `*LauncherExecution`:
```go
type LauncherExecution struct {
    Launcher     *component.LauncherV2        // The launcher instance
    MockFS       *component.MockFileSystem    // Mock file system
    MockCmd      *component.MockCommandExecutor // Mock command executor
    MockObjStore *component.MockObjectStoreClient // Mock object store
    Task         *apiv2beta1.PipelineTaskDetail // Updated task after execution
}
```

### What RunLauncher Does

1. **Gets the task** created by the driver
2. **Reuses ExecutorInput** from `execution.ExecutorInput`
3. **Gets component/task specs** from TestContext's ScopePath
4. **Creates launcher** with real LauncherV2 code
5. **Injects mocks** for file system, command executor, and object store
6. **Pre-populates** input artifacts in mock object store
7. **Executes** the launcher's full flow (download inputs → execute command → collect outputs → upload artifacts)
8. **Returns** LauncherExecution with mocks available for verification

### Mock Configuration

```go
// Configure output files that the component "writes"
launcherExec := tc.RunLauncher(execution, map[string][]byte{
    "/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
    "/tmp/outputs/metric": []byte("0.95"),
})

// Verify command was executed
assert.Equal(t, 1, launcherExec.MockCmd.CallCount())

// Verify artifacts were uploaded
uploads := launcherExec.MockObjStore.GetUploadCallsForKey("model")
assert.Len(t, uploads, 1)
assert.Equal(t, "s3://bucket/output/model.pkl", uploads[0].RemoteURI)
```

## Mock Infrastructure

### MockAPI

Provides in-memory KFP API v2beta1 operations:

```go
mockAPI := kfpapi.NewMockAPI()

// Automatically used by TestContext
tc := NewTestContextWithRootExecuted(t, runtimeConfig, pipelinePath)

// Access via TestContext
tc.MockAPI.GetRun(ctx, &apiv2beta1.GetRunRequest{RunId: runID})
```

Key features:
- Automatic task/artifact hydration (tasks populated with their artifacts)
- Support for CreateTask, UpdateTask, GetTask, ListTasks
- Support for CreateArtifact, CreateArtifactTask
- Filter support for cache lookups

### MockFileSystem

Simulates file system operations:

```go
mockFS := component.NewMockFileSystem()

// Set file contents
mockFS.SetFileContent("/tmp/outputs/metric", []byte("0.95"))

// Verify operations
assert.Len(t, mockFS.ReadFileCalls, 1)
assert.Equal(t, "/tmp/outputs/metric", mockFS.ReadFileCalls[0])
```

### MockCommandExecutor

Simulates command execution:

```go
mockCmd := component.NewMockCommandExecutor()

// Configure success/failure
mockCmd.RunError = nil  // or fmt.Errorf("command failed")

// Verify execution
assert.Equal(t, 1, mockCmd.CallCount())
assert.Equal(t, "python", mockCmd.RunCalls[0].Cmd)
```

### MockObjectStoreClient

Simulates object store operations:

```go
mockObjStore := component.NewMockObjectStoreClient()

// Pre-populate input artifacts (done automatically by RunLauncher)
mockObjStore.SetArtifact("s3://bucket/input/data.csv", []byte("data"))

// Verify uploads
uploads := mockObjStore.GetUploadCallsForKey("model")
assert.Len(t, uploads, 1)
```

## Complete Example

Here's a complete test demonstrating the full pattern:

```go
func TestArtifactPassing(t *testing.T) {
    // 1. Create test context with root DAG executed
    tc := NewTestContextWithRootExecuted(
        t,
        &pipelinespec.PipelineJob_RuntimeConfig{},
        "test_data/artifact_pipeline.yaml",
    )

    // 2. Run driver for producer task
    producerExecution, producerTask := tc.RunContainer(
        "create-dataset",
        tc.RootTask,
        nil,  // not in a loop
        true, // auto-update scope
    )

    // Verify driver created ExecutorInput with output artifact
    require.NotNil(t, producerExecution.ExecutorInput.Outputs)
    require.Contains(t, producerExecution.ExecutorInput.Outputs.Artifacts, "output_dataset")

    // 3. Run launcher to simulate component execution
    producerLauncherExec := tc.RunLauncher(producerExecution, map[string][]byte{
        "/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
    })

    // Verify launcher executed command
    require.Equal(t, 1, producerLauncherExec.MockCmd.CallCount())

    // Verify launcher uploaded output artifact
    require.Len(t, producerLauncherExec.Task.Outputs.Artifacts, 1)
    outputArtifactID := producerLauncherExec.Task.Outputs.Artifacts[0].Artifacts[0].ArtifactId

    // 4. Run driver for consumer task
    consumerExecution, consumerTask := tc.RunContainer(
        "process-dataset",
        tc.RootTask,
        nil,
        true,
    )

    // Verify driver resolved input artifact from producer
    require.Contains(t, consumerExecution.ExecutorInput.Inputs.Artifacts, "input_dataset")
    inputArtifacts := consumerExecution.ExecutorInput.Inputs.Artifacts["input_dataset"].Artifacts
    require.Len(t, inputArtifacts, 1)
    require.Equal(t, outputArtifactID, inputArtifacts[0].ArtifactId)

    // 5. Run launcher for consumer
    consumerLauncherExec := tc.RunLauncher(consumerExecution, map[string][]byte{
        "/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
    })

    // Verify launcher downloaded input artifact
    require.Len(t, consumerLauncherExec.MockObjStore.DownloadCalls, 1)
    require.Equal(t, "input_dataset", consumerLauncherExec.MockObjStore.DownloadCalls[0].ArtifactKey)
}
```

## Loop Testing Example

```go
func TestLoopArtifacts(t *testing.T) {
    tc := NewTestContextWithRootExecuted(
        t,
        &pipelinespec.PipelineJob_RuntimeConfig{},
        "test_data/loop_pipeline.yaml",
    )

    // Run loop DAG driver
    loopExecution, loopTask := tc.RunDag("for-loop-1", tc.RootTask)

    // Run iterations
    for index := 0; index < 3; index++ {
        // Run driver for iteration
        iterExecution, iterTask := tc.RunContainer(
            "process-item",
            loopTask,
            util.Int64Pointer(int64(index)),
            true, // auto-update scope
        )

        // Verify iteration index is in ExecutorInput
        require.NotNil(t, iterExecution.ExecutorInput)

        // Run launcher for iteration
        iterLauncherExec := tc.RunLauncher(iterExecution, map[string][]byte{
            "/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
        })

        // Verify execution
        require.Equal(t, 1, iterLauncherExec.MockCmd.CallCount())

        // Get artifact ID for DAG output collection
        artifactID := iterLauncherExec.Task.Outputs.Artifacts[0].Artifacts[0].ArtifactId

        // Mock DAG artifact collection (will be added to launcher later)
        tc.MockLauncherArtifactTaskCreate(
            "process-item",
            loopExecution.TaskID,
            "output_artifact",
            artifactID,
            util.Int64Pointer(int64(index)),
            apiv2beta1.IOType_ITERATOR_OUTPUT,
        )
    }

    // Exit loop DAG scope
    tc.ExitDag()

    // Verify loop collected all iteration outputs
    loopTask, err := tc.ClientManager.KFPAPIClient().GetTask(
        context.Background(),
        &apiv2beta1.GetTaskRequest{TaskId: loopExecution.TaskID},
    )
    require.NoError(t, err)
    require.Len(t, loopTask.Outputs.Artifacts, 3)
}
```

## Mock Output Creation Helpers

For simulating launcher-created outputs:

### Parameters

```go
// Mock launcher creating an output parameter
tc.MockLauncherOutputParameterCreate(
    taskID,
    "output_param_key",
    structpb.NewStringValue("output_value"),
    apiv2beta1.IOType_OUTPUT,
    "task-name",
    iterationIndex,  // nil for non-loop tasks
)
```

### Artifacts

```go
// Mock launcher creating an output artifact
artifactID := tc.MockLauncherOutputArtifactCreate(
    taskID,
    "artifact_key",
    apiv2beta1.Artifact_Dataset,
    apiv2beta1.IOType_OUTPUT,
    "task-name",
    iterationIndex,  // nil for non-loop tasks
)
```

### DAG Artifact Collection

```go
// Mock launcher propagating artifact to parent DAG
// (This will be added to real launcher later)
tc.MockLauncherArtifactTaskCreate(
    producerTaskName,
    dagTaskID,
    artifactKey,
    artifactID,
    iterationIndex,
    apiv2beta1.IOType_OUTPUT,  // or ITERATOR_OUTPUT for loops
)
```

## Best Practices

1. **Always use RunLauncher after RunContainer**: The driver creates ExecutorInput, the launcher consumes it
2. **Verify driver outputs**: Check that ExecutorInput contains expected inputs/outputs
3. **Verify launcher behavior**: Use mock verification to ensure correct execution
4. **Use autoUpdateScope=true for containers**: Simplifies scope management
5. **Call ExitDag() after DAG children**: Required for proper scope tracking
6. **Use RefreshRun() before assertions**: Ensures you have latest task state
7. **Keep mock DAG collection calls**: Until artifact propagation is added to launcher

## Common Pitfalls

1. **Forgetting to ExitDag()**: Causes scope path to be incorrect for subsequent tasks
2. **Not using execution.ExecutorInput**: Recreating ExecutorInput defeats the purpose of driver testing
3. **Wrong iteration index**: Must match the index used in RunContainer
4. **Stale task data**: Call RefreshRun() or GetTask() to get latest state
5. **Missing output files**: Launcher needs output_metadata.json file at minimum

## Future Enhancements

- **DAG artifact propagation in launcher**: MockLauncherArtifactTaskCreate calls will be replaced by real launcher logic
- **Automatic output discovery**: Launcher will traverse DAG hierarchy to propagate artifacts
- **Enhanced cache testing**: More comprehensive cache key validation
