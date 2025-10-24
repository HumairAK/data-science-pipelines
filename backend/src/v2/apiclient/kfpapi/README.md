# KFP API Package

This package provides a clean abstraction over the KFP v2beta1 API services for use by drivers, launchers, and other components.

## Overview

The `kfpapi` package contains:

- **`API` interface**: A minimal interface exposing KFP API operations needed by drivers and launchers
- **`clientAdapter`**: Production implementation that wraps the gRPC clients
- **`MockAPI`**: Test implementation for unit testing

## Migration from `driver/common`

This package was previously located in `backend/src/v2/driver/common` as `DriverAPI`. It has been moved to a shared location since it's used by both drivers and launchers (not driver-specific).

### Breaking Changes

- **Package location**: Moved from `driver/common` to `apiclient/kfpapi`
- **Interface name**: `DriverAPI` → `API` (more generic)
- **Constructor**: `NewDriverAPI()` → `New()`
- **Mock implementation**: `MockDriverAPI` → `MockAPI`
- **Mock constructor**: `NewMockDriverAPI()` → `NewMockAPI()`

### Backward Compatibility

For backward compatibility, `driver/common/api.go` still exports type aliases:

```go
type DriverAPI = kfpapi.API
func NewDriverAPI(c *apiclient.Client) DriverAPI { ... }
```

**Migration recommendation**: Update imports to use `apiclient/kfpapi` directly. The compatibility layer may be removed in a future version.

## Usage

### Production Code

```go
import (
    "github.com/kubeflow/pipelines/backend/src/v2/apiclient"
    "github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
)

// Create gRPC client
cfg := apiclient.FromEnv()
client, err := apiclient.New(cfg)
if err != nil {
    // handle error
}

// Wrap in API interface
api := kfpapi.New(client)

// Use the API
run, err := api.GetRun(ctx, &GetRunRequest{RunId: "..."})
```

### Test Code

```go
import (
    "github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
)

// Create mock
mockAPI := kfpapi.NewMockAPI()

// Setup test data
mockAPI.AddRun(testRun)
mockAPI.AddPipelineVersion("pipeline-id", "version-id", testVersion)

// Use in tests
run, err := mockAPI.GetRun(ctx, &GetRunRequest{RunId: testRun.RunId})
```

## API Operations

The `API` interface provides the following operations:

### Run Operations
- `GetRun` - Retrieve a run by ID

### Task Operations
- `CreateTask` - Create a new task
- `UpdateTask` - Update an existing task
- `GetTask` - Retrieve a task by ID
- `ListTasks` - List tasks with filtering

### Artifact Operations
- `CreateArtifact` - Create a new artifact
- `ListArtifactsByURI` - List artifacts by URI
- `ListArtifactTasks` - List artifact-task relationships
- `CreateArtifactTask` - Create an artifact-task relationship
- `CreateArtifactTasks` - Bulk create artifact-task relationships

### Pipeline Operations
- `GetPipelineVersion` - Retrieve a pipeline version
- `FetchPipelineSpecFromRun` - Get pipeline spec from a run (either embedded or via version reference)

## Design Principles

1. **Minimal surface area**: Only expose operations actually needed by drivers/launchers
2. **Testability**: Provide mock implementation for unit testing
3. **Abstraction**: Hide gRPC client details behind a simple interface
4. **Evolvability**: Allow underlying client changes without affecting consumers
