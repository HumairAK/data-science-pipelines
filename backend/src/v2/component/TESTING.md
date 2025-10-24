# Testing LauncherV2 Components

This document explains how to test LauncherV2 components using the new mock-based testing infrastructure.

## Overview

The `LauncherV2` has been refactored to support dependency injection, making it easy to write component-level tests without requiring actual file system operations, command execution, or object store access.

## Architecture

### Dependency Interfaces

The launcher now depends on three key interfaces instead of directly calling `os`, `exec`, and `objectstore`:

1. **FileSystem** - Abstracts file/directory operations
   - `MkdirAll()` - Create directories
   - `Create()` - Create files
   - `ReadFile()` - Read file contents
   - `WriteFile()` - Write file contents
   - `Stat()` - Get file info

2. **CommandExecutor** - Abstracts command execution
   - `Run()` - Execute a command with stdin/stdout/stderr

3. **ObjectStoreClient** - Abstracts artifact storage
   - `UploadArtifact()` - Upload artifacts to remote storage
   - `DownloadArtifact()` - Download artifacts from remote storage

### Mock Implementations

For testing, we provide three mock implementations:

- `MockFileSystem` - In-memory file system with call tracking
- `MockCommandExecutor` - Configurable command execution with call tracking
- `MockObjectStoreClient` - In-memory object store with call tracking

All mocks are thread-safe and provide helper methods for assertions.

## Usage

### Basic Pattern

```go
func TestMyComponent(t *testing.T) {
    // 1. Create mock KFP API
    mockAPI := kfpapi.NewMockAPI()

    // 2. Setup test data (run, task, etc.)
    run := &apiv2beta1.Run{ /* ... */ }
    mockAPI.AddRun(run)

    // 3. Create launcher
    launcher, err := NewLauncherV2(executorInputJSON, cmdArgs, opts, clientManager)
    require.NoError(t, err)

    // 4. Setup mocks
    mockFS := NewMockFileSystem()
    mockCmd := NewMockCommandExecutor()
    mockObjStore := NewMockObjectStoreClient()

    // 5. Configure mock behavior
    mockFS.SetFileContent("/tmp/output.txt", []byte("result"))
    mockCmd.RunError = nil
    mockObjStore.SetArtifact("s3://bucket/input.csv", []byte("data"))

    // 6. Inject mocks
    launcher.WithFileSystem(mockFS).
        WithCommandExecutor(mockCmd).
        WithObjectStore(mockObjStore)

    // 7. Execute
    _, err = launcher.execute(ctx, "python", []string{"script.py"})
    require.NoError(t, err)

    // 8. Verify behavior
    assert.Equal(t, 1, mockCmd.CallCount())
    assert.Len(t, mockObjStore.UploadCalls, 1)
}
```

### Testing Artifact Handling

```go
func TestArtifactDownloadAndUpload(t *testing.T) {
    mockObjStore := NewMockObjectStoreClient()

    // Setup input artifact
    mockObjStore.SetArtifact("s3://bucket/input/data.csv", []byte("training,data"))

    // Test download
    err := mockObjStore.DownloadArtifact(ctx, "s3://bucket/input/data.csv", "/local/data.csv", "input_data")
    require.NoError(t, err)

    // Verify download was called correctly
    assert.Len(t, mockObjStore.DownloadCalls, 1)
    assert.Equal(t, "input_data", mockObjStore.DownloadCalls[0].ArtifactKey)
    assert.Equal(t, "s3://bucket/input/data.csv", mockObjStore.DownloadCalls[0].RemoteURI)

    // Test upload
    err = mockObjStore.UploadArtifact(ctx, "/local/model.pkl", "s3://bucket/output/model.pkl", "model")
    require.NoError(t, err)

    // Query uploads by artifact key
    modelUploads := mockObjStore.GetUploadCallsForKey("model")
    assert.Len(t, modelUploads, 1)
    assert.Equal(t, "s3://bucket/output/model.pkl", modelUploads[0].RemoteURI)
}
```

### Testing Command Execution

```go
func TestCommandWithCustomBehavior(t *testing.T) {
    mockCmd := NewMockCommandExecutor()

    // Setup custom behavior
    mockCmd.RunFunc = func(ctx context.Context, cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
        // Simulate successful execution with output
        stdout.Write([]byte("Training completed\n"))
        stdout.Write([]byte("Accuracy: 0.95\n"))
        return nil
    }

    // Execute
    var stdout bytes.Buffer
    err := mockCmd.Run(ctx, "python", []string{"train.py"}, nil, &stdout, nil)

    // Verify
    require.NoError(t, err)
    assert.Contains(t, stdout.String(), "Accuracy: 0.95")
    assert.Equal(t, "python", mockCmd.RunCalls[0].Cmd)
}
```

### Testing File System Operations

```go
func TestFileOperations(t *testing.T) {
    mockFS := NewMockFileSystem()

    // Test directory creation
    err := mockFS.MkdirAll("/tmp/outputs", 0755)
    require.NoError(t, err)

    // Test file writing
    err = mockFS.WriteFile("/tmp/outputs/metrics.json", []byte(`{"acc": 0.95}`), 0644)
    require.NoError(t, err)

    // Test file reading
    content, err := mockFS.ReadFile("/tmp/outputs/metrics.json")
    require.NoError(t, err)
    assert.Equal(t, `{"acc": 0.95}`, string(content))

    // Verify operations
    assert.Len(t, mockFS.MkdirAllCalls, 1)
    assert.Equal(t, "/tmp/outputs", mockFS.MkdirAllCalls[0].Path)
}
```

### Testing KFP API Interactions

```go
func TestTaskUpdates(t *testing.T) {
    mockAPI := kfpapi.NewMockAPI()

    // Create test data
    run := &apiv2beta1.Run{
        RunId: "run-123",
        State: apiv2beta1.RuntimeState_RUNNING,
        PipelineSource: &apiv2beta1.Run_PipelineSpec{
            PipelineSpec: &structpb.Struct{},
        },
    }
    mockAPI.AddRun(run)

    // Create task
    task := &apiv2beta1.PipelineTaskDetail{
        TaskId: "task-456",
        RunId:  "run-123",
        Status: apiv2beta1.PipelineTaskDetail_RUNNING,
    }
    _, err := mockAPI.CreateTask(ctx, &apiv2beta1.CreateTaskRequest{Task: task})
    require.NoError(t, err)

    // Update task
    task.Status = apiv2beta1.PipelineTaskDetail_SUCCEEDED
    _, err = mockAPI.UpdateTask(ctx, &apiv2beta1.UpdateTaskRequest{
        TaskId: "task-456",
        Task:   task,
    })
    require.NoError(t, err)

    // Verify
    updatedTask, err := mockAPI.GetTask(ctx, &apiv2beta1.GetTaskRequest{TaskId: "task-456"})
    require.NoError(t, err)
    assert.Equal(t, apiv2beta1.PipelineTaskDetail_SUCCEEDED, updatedTask.Status)
}
```

## Mock Features

### MockFileSystem

**In-memory storage:**
- Files and directories are stored in memory maps
- No actual file system interaction

**Call tracking:**
- `MkdirAllCalls` - Track all directory creations
- `CreateCalls` - Track all file creations
- `ReadFileCalls` - Track all file reads
- `WriteFileCalls` - Track all file writes
- `StatCalls` - Track all stat calls

**Helper methods:**
- `SetFileContent(name, data)` - Pre-populate file content
- `GetFileContent(name)` - Retrieve file content for assertions

**Error injection:**
- Set `MkdirAllError`, `ReadFileError`, etc. to simulate failures

### MockCommandExecutor

**Call tracking:**
- `RunCalls` - Track all command executions with cmd, args

**Custom behavior:**
- `RunFunc` - Provide custom function for execution logic
- `RunError` - Set error to return

**Helper methods:**
- `CallCount()` - Get number of times Run was called

### MockObjectStoreClient

**In-memory storage:**
- Artifacts stored in memory map (URI -> data)

**Call tracking:**
- `UploadCalls` - Track all uploads (LocalPath, RemoteURI, ArtifactKey)
- `DownloadCalls` - Track all downloads (LocalPath, RemoteURI, ArtifactKey)

**Helper methods:**
- `SetArtifact(uri, data)` - Pre-populate artifact
- `WasUploaded(uri)` - Check if artifact was uploaded
- `GetUploadCallsForKey(key)` - Get uploads for specific artifact key
- `GetDownloadCallsForKey(key)` - Get downloads for specific artifact key

**Error injection:**
- `UploadError` - Set error to return on upload
- `DownloadError` - Set error to return on download

## Best Practices

1. **Use specific assertions**: Instead of just checking call counts, verify the actual parameters passed

2. **Test failure paths**: Use error injection to test error handling
   ```go
   mockFS.ReadFileError = os.ErrNotExist
   _, err := launcher.collectOutputParameters(executorOutput)
   assert.Error(t, err)
   ```

3. **Test artifact tracking**: Verify correct artifacts are uploaded/downloaded
   ```go
   modelUploads := mockObjStore.GetUploadCallsForKey("model_output")
   assert.Len(t, modelUploads, 1)
   assert.Equal(t, "s3://bucket/models/model.pkl", modelUploads[0].RemoteURI)
   ```

4. **Verify task status updates**: Check that tasks are updated correctly
   ```go
   updatedTask, _ := mockAPI.GetTask(ctx, &apiv2beta1.GetTaskRequest{TaskId: taskID})
   assert.Equal(t, apiv2beta1.PipelineTaskDetail_SUCCEEDED, updatedTask.Status)
   ```

5. **Test command execution**: Verify correct commands are executed with correct arguments
   ```go
   assert.Equal(t, "python", mockCmd.RunCalls[0].Cmd)
   assert.Equal(t, []string{"train.py", "--input", "data.csv"}, mockCmd.RunCalls[0].Args)
   ```

## Examples

See `launcher_v2_example_test.go` for complete working examples demonstrating:
- Full component testing with all mocks
- Artifact handling patterns
- Command execution testing
- File system operation testing
- KFP API interaction testing

## Migration Guide

If you have existing tests using real file system/command execution:

**Before:**
```go
// Test that actually creates files and runs commands
cmd := exec.Command("python", "script.py")
err := cmd.Run()
data, _ := os.ReadFile("/tmp/output.txt")
```

**After:**
```go
mockCmd := NewMockCommandExecutor()
mockFS := NewMockFileSystem()
mockFS.SetFileContent("/tmp/output.txt", []byte("result"))

launcher.WithCommandExecutor(mockCmd).WithFileSystem(mockFS)
_, err := launcher.execute(ctx, "python", []string{"script.py"})

assert.Equal(t, 1, mockCmd.CallCount())
data, _ := mockFS.GetFileContent("/tmp/output.txt")
```

Benefits:
- ✅ No actual file I/O
- ✅ No actual command execution
- ✅ Fast and deterministic
- ✅ Easy to test error conditions
- ✅ Full control over dependencies
