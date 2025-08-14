# Kubeflow Pipelines ML-Metadata (MLMD) Removal Proposal

## Proposed Solution

[//]: # (TODO: add)

### New KFP Database Schema

See [schema_changes.sql](./schema_changes.sql) for the database schema additions and changes. 

Note that a task is a db model for a task node type as viewed in the Run Graph of the UI.

### KFP Server API

To facilitate the removal of MLMD, the KFP server will now take on the burden of handling Artifacts, Dags, input resolution, and so on. 

See [artifacts.proto](./protos/artifacts.proto) for the Artifact server changes. 

Instead of an MLMD client, the Driver and Launcher will include and leverage a `v2beta1.RunServiceClient` retrieved via `NewRunServiceClient()` in `backend/api/v2beta1`.

See [runs.proto](./protos/runs.proto) for additions to the RunService client.

Example [runs.json](./protos/runs.json) to see what the updated run response might look like, some existing feels are 
omitted for clarity.

### Driver changes

The driver makes various interactions with MLMD which will need to be adjusted. The main transition will be from creating Executions to Tasks.

The `ROOT_DAG` driver will pass `execution_id` flag to succeeding drivers within the pipeline. These will need to be updated to instead pass `parent_task_id`. 

There are various control flows to consider that create and manage executions. We'll address them individually below.

#### Control Flows 

In KFP the driver component is responsible for creating new Executions. There are two types of driver executions: 

* ContainerExecution - always precedes the launcher/executor pod
  * Makes caching decisions
  * Resolves inputs 
  * Builds podspec for the executor pod
* DagExecution - this can be further subdivided to:
  * RootDag - runs once per pipeline
    * Creates the PipelineRun context in MLMD  
    * Contains runtime inputs information
  * Dag - runs one or more times per task group (Condition, ConditionBranch, Loop, LoopIteration)
    * Resolves conditions for conditional dags 
    * Resolves inputs 
    * Resolves iteration counts (for loops)

Each execution type interacts with mlmd in slightly different ways. 


### MLMD Client replacement 

MLMD client will need to be replaced. These are the relevant calls used by driver and launcher:

```go
package metadata

type DAG struct {
  Execution *Execution
}
type Execution struct {
  execution *pb.Execution
  pipeline  *Pipeline
}
// A pipeline context contains: create/update time, namespace, pipeline_root
// The pipelineCtx represents a Pipeline (not PipelineVersion) and is created once per pipeline
// This struct is primarily used for creating execution Associations and can most likely be discarded
type Pipeline struct {
  pipelineCtx    *pb.Context
  pipelineRunCtx *pb.Context
}
type InputArtifact struct {
  Artifact *pb.Artifact
}
type OutputArtifact struct {
  Name     string
  Artifact *pb.Artifact
  Schema   string
}

// GetPipeline actually gets or creates a context if it doesn't already exist for this pipeline and pipelineRun (one context each)
func (c *Client) GetPipeline(ctx context.Context, pipelineName, runID, namespace, runResource, pipelineRoot, storeSessionInfo string) (*Pipeline, error)
func (c *Client) GetDAG(ctx context.Context, executionID int64) (*DAG, error)
func (c *Client) PublishExecution(ctx context.Context, execution *Execution, outputParameters map[string]*structpb.Value, outputArtifacts []*OutputArtifact, state pb.Execution_State) error
func (c *Client) CreateExecution(ctx context.Context, pipeline *Pipeline, config *ExecutionConfig) (*Execution, error)
// Creates execution, updating it with pod and status info
func (c *Client) PrePublishExecution(ctx context.Context, execution *Execution, config *ExecutionConfig) (*Execution, error)
func (c *Client) UpdateDAGExecutionsState(ctx context.Context, dag *DAG, pipeline *Pipeline) error
func (c *Client) PutDAGExecutionState(ctx context.Context, executionID int64, state pb.Execution_State) error
func (c *Client) GetExecutions(ctx context.Context, ids []int64) ([]*pb.Execution, error)
func (c *Client) GetExecution(ctx context.Context, id int64) (*Execution, error)
func (c *Client) GetPipelineFromExecution(ctx context.Context, id int64) (*Pipeline, error)
func (c *Client) GetExecutionsInDAG(ctx context.Context, dag *DAG, pipeline *Pipeline, filter bool) (executionsMap map[string]*Execution, err error)

func (c *Client) GetEventsByArtifactIDs(ctx context.Context, artifactIds []int64) ([]*pb.Event, error)
func (c *Client) GetArtifactName(ctx context.Context, artifactId int64) (string, error) // Not used
func (c *Client) GetArtifacts(ctx context.Context, ids []int64) ([]*pb.Artifact, error)
func (c *Client) GetOutputArtifactsByExecutionId(ctx context.Context, executionId int64) (map[string]*OutputArtifact, error)
func (c *Client) GetInputArtifactsByExecutionID(ctx context.Context, executionID int64) (inputs map[string]*pipelinespec.ArtifactList, err error)
func (c *Client) RecordArtifact(ctx context.Context, outputName, schema string, runtimeArtifact *pipelinespec.RuntimeArtifact, state pb.Artifact_State, bucketConfig *objectstore.Config) (*OutputArtifact, error)
func (c *Client) GetOrInsertArtifactType(ctx context.Context, schema string) (typeID int64, err error)
func (c *Client) FindMatchedArtifact(ctx context.Context, artifactToMatch *pb.Artifact, pipelineContextId int64) (matchedArtifact *pb.Artifact, err error)
```

These will be replaced calls to v2beta1.RunService instead:

```go
package run_client

// Replaces GetPipeline, additionally we will need to pass experiment ID to the driver/launcher
// It also replaces GetPipelineFromExecution, since Tasks have a RunID
func (c *RunServerClient) GetRun(ctx, runID, experimentID) (*apiv2beta1.Run, error)
// Replaces GetDAG (filter on task type), GetExecutions, GetExecution
func (c *RunServerClient) GetTask(ctx, taskID) (*apiv2beta1.PipelineTaskDetail, error)  // uses GetTask() in RunsAPI
func (c *RunServerClient) GetTasks(ctx, taskID) ([]*apiv2beta1.PipelineTaskDetail, error) // uses ListTasks() in RunsAPI

// Replaces PublishExecution, CreateExecution, PrePublishExecution, UpdateDAGExecutionsState
func (c *RunServerClient) CreateTask(ctx context.Context, task apiv2beta1.PipelineTaskDetail) (*apiv2beta1.PipelineTaskDetail, error)
func (c *RunServerClient) UpdateTask(ctx context.Context, task apiv2beta1.PipelineTaskDetail) (*apiv2beta1.PipelineTaskDetail, error)

// Replaces GetExecutionsInDAG
// Use Run API's ListTasks() with run_id field 
func (c *RunServerClient) GetChildTasks(ctx context.Context, task apiv2beta1.PipelineTaskDetail) (map[string]*apiv2beta1.PipelineTaskDetail, error)
```

In a similar manner, the v2beta1 ArtifactService can be used to implement the following: 

* `GetEventsByArtifactIDs` rename to `GetArtifactEventsByArtifactIDs`, via `ListArtifactEvents`
* `GetArtifacts`, via `ListArtifacts`
* `RecordArtifact`, via `CreateArtifact`
* `GetOutputArtifactsByExecutionId` rename to `GetOutArtifactsByTaskID`, via `ListArtifactEvents` and `ListArtifacts`
* `GetInputArtifactsByExecutionID` rename to `GetInputArtifactsByTaskID`, via `ListArtifactEvents` and `ListArtifacts`
* `GetOrInsertArtifactType`, use a combination of `GetArtifact`, `UpdateArtifact`
* `FindMatchedArtifact`, use `ListArtifacts` and filter on `uri`


##### ContainerExecution

Container drivers will now create a task of type `RUNTIME`. When creating the PodSpecPatch, we will need to pass the `task_id` instead of the `execution_id` flag.

##### Loops 

Loops in KFP today require two types of dags, there's either a dag that has an `iteration_count` or a dag that has an `iteration_index`. 
We'll refer to these as Loop and LoopIteration respectively. A `Loop` is a task grouping of components that will run within a loop and tracks the total count of iterations via `iteration_count`. A `LoopIteration` is a dag that tracks the current iteration for a given loop via `iteration_index`.

Each of these results in an `DagExecution`. The components that run for each iteration will also have their regular `ContainerExecutions`.
Each of these is used to resolve inputs/outputs and will need to be logged as Tasks into the Tasks table. 

Instead of these executions, will be switching to creating Tasks of types `LOOP` and `LOOP_ITERATION` respectively, and leverage the RunServer and ArtifactServer for input resolution. 

##### Exit handler 

Any task under the `dsl.Exithandler` group falls within a Dag execution. These tasks will now be grouped under a task of type `EXIT_HANDLER`.

##### DSL If/Else/ElseIf

When working with Conditions in KFP, new nodes are introduced in the Pipeline Graph, they are prefixed with `condition-` 
or `condition-branches-`. 

1. **`CONDITION`** - Represents a conditional task group (If/Else/ElseIf), in KFP it is represented by a DAG driver that outputs a condition parameter which determines whether the underlying dag or components should execute.

2. **`CONDITION_BRANCHES`** - Represents the branches that stem from a conditional statement.

Each of these results in a new dag execution. Instead of these executions, will be switching to creating Tasks of types `CONDITION_BRANCH` and `CONDITION` respectively, and leverage the RunServer and ArtifactServer for input resolution.

##### Caching

Caching mechanisms should remain the same. The `PipelineTaskDetail` proto will support a `cache_fingerprint` field. 
For task creation and updates this field can be provided for `CreateTask`, `UpdateTask`.

### Launcher changes 

* Status updates and reporting for task level

### Nested Pipelines 

There is no way direct way to detect whether a driver run is for a Nested execution, to accommodate there is a generic `DAG` task type is provided to fit such cases.
Alternatively, we could provide an SDK update to declare a task type in a field on a `ComponentSpec` `dag` field.

### StoreSessionInfo 

Currently, Artifact credential info is stored as a custom property, and is called `store_session_info`. In this proposal, we will not port over this capability as a 
custom artifact property, and we will instead remove this from the `rood_dag.go`: 

```go
storeSessionInfo, err = cfg.GetStoreSessionInfo(pipelineRoot)
```

And build it use it directly in `launcher_v2.go`, replacing: 

```go
storeSessionInfo, err := objectstore.GetSessionInfoFromString(execution.GetPipeline().GetStoreSessionInfo())
```

### Metrics 

Metrics in KFP today are stored as Artifacts, they have the following Artifact Types: 

* system.Metrics - Regular Key -> NumberValue pair 
* system.ClassificationMetrics - Key -> JSON 
* system.SlicedClassificationMetrics -> Key -> JSON

The values for these Metrics Artifacts are stored as `CustomProperties`, they aren't actually stored in object store. 
So it is questionable that they are treated as Artifacts to begin with. Instead of porting this behavior, we'll instead leverage the Metrics table in KFP which is currently unused. 

We will log the Metrics there when such artifact types are encountered in the launcher. These can be addressed in `launcher_v2.go` when `uploadOutputArtifact` is called. During this invocation we can check for an artifacts type via: 

```go
	schemaTitle := runtimeArtifact.Type.GetSchemaTitle()
	switch schemaTitle {
	case "system.Metrics":  // Handles Metric type, do something similar for ClassificationMetrics & SlicedClassificationMetrics
		err := LogMetric(...)
		...
    case "system.Artifact":
        err := RecordArtifact()
		...
```

In the executor Input we can abstain from storing a URI since this does not apply to Metrics.

When the driver is looking to resolve Artifacts, to store in the ExecutorInput, it will need to ensure it's differentiating between Metrics and other Artifact types.

To keep the Python SDK will continue to interpret Metrics as artifacts, this helps maintain backwards compatibility. The Driver will need to ensure when it is creating the Artifacts list during the call to `resolveInputs -> resolveInputArtifact -> resolveUpstreamArtifacts() -> artifact.ToRuntimeArtifact()`, we are parsing Metrics. The updated pseudocode in `resolveUpstreamArtifacts` will be something like: 

```go
package driver

func resolveUpstreamArtifacts(cfg resolveUpstreamOutputsConfig) (*pipelinespec.ArtifactList, error) {
  for {
    ...
  } else {
    // use the Component *pipelinespec.ComponentSpec.ComponentInputsSpec from Options in driver.go to determine 
	// artifact schema type, 
    schemaTitle := determineArtifactSchema(ComponentInputSpec, TaskSpec)
    switch schemaTitle {
    case "system.Metrics":  // Handles Metric type, do something similar for ClassificationMetrics & SlicedClassificationMetrics
	  // GetOutputMetricsByTaskID can fetch the Task via GetTask (if we don't already have the task), 
	  // and can parse the `output_metrics` to return map[string]*OutputArtifact or just the *OutputArtifact
      outputs, err := GetOutputMetricsByTaskID(cfg.ctx, taskID)
    case "system.Artifact":
      outputs, err := GetOutArtifactsByTaskID(cfg.ctx, taskID)
  }
}
```

### Frontend Changes 
For run details, MLMD data is fetched in `RuntimeNodeDetailsV2.tsx` via: 

```typescript
const context = await getKfpV2RunContext(runId);
const executions = await getExecutionsFromContext(context);
const artifacts = await getArtifactsFromContext(context);
const events = await getEventsByExecutions(executions);
```

These can be replaced by the following new implementations: 

```typescript
// context no longer needed, use the Run object which often readily available wherever context is required
const tasks = run.run_details.task_details;  // run is a V2beta1Run
const artifacts = await fetchArtifactsFromTasks(tasks); // the information is now available in the task.
// a separate call for this is may not needed, as the required info may already be present in `tasks`
const events = await getArtifactEventsByTasks(tasks); // uses ListArtifactEvents()
```

The `Visualization` Nav in `RuntimeNodeDetailsV2.tsx` will also need to be updated to take `Metrics` fetch from `Task` proto, instead of `Artifacts` from MLMD.

The Artifact Node in the UI should also no longer display an `Artifact URI`, as this is not applicable to metrics. 

The `CompareV2.tsx` also makes various calls to MLMD, much like `RuntimeNodeDetailsV2`: 

```typescript
      Promise.all(
        runIds.map(async runId => {
          // TODO(zijianjoy): MLMD query is limited to 100 artifacts per run.
          // https://github.com/google/ml-metadata/blob/5757f09d3b3ae0833078dbfd2d2d1a63208a9821/ml_metadata/proto/metadata_store.proto#L733-L737
          const context = await getKfpV2RunContext(runId);
          const executions = await getExecutionsFromContext(context);
          const artifacts = await getArtifactsFromContext(context);
          const events = await getEventsByExecutions(executions);
          return {
            executions,
            artifacts,
            events,
          } as MlmdPackage;
        }),
      ),
```

This and associated code will also need to be updated to leverage Tasks retrieved via the Runs API server.

#### Run Reporting

The Persistent Agent calls the KFP Server's [report_server.go](../../backend/src/apiserver/server/report_server.go) for updating the Run in DB. This includes updates to the Task db.

Because we are relying on driver/launcher to create/update tasks, we will no longer require Persistent Agent to report on task details, we will need to get rid of this portion of the code.
This is the key piece of code from `report-server.go`:

```go
_, err = s.reportTasksFromExecution(newExecSpec, runId)
```

### Auth Considerations 

The driver/launcher will be making requests for tasks, artifacts, etc. from API server 
How will they check for authorization on behalf of user that they have access to this namespace? 

When driver is creating an artifact, it always creates it in the namespace the pipeline is running, so it's fine if 
Driver has scope to create/fetch artifacts in that namespace, since the run was gated already via namespace. 

What about importing artifacts? 

Driver Launcher does SAR using Pipeline Service Account - consider importer

### Delivery Plan 

* Add the tables and gorm model changes
* Start logging all Dags and as tasks alongside MLMD 
    * Do each control flow seperately?
    * Should also update the task details proto/json reporting on pipeline server
* Update the API server to display: 
  * Accurate Task pod name 
  * Child tasks 
* Update the UI to start reading from the API server, removing MLMD logic
* Update resolve/input logic to use Task details and remove all MLMD invocations from the backend
* Keep MLMD around for the next release, and add migration scripts/code 
* In the following release, remove MLMD manifests, deployment options, etc. 

### Migration

For tasks/pipelineTaskDetail if we're re-using the old table then migration will become tricky 
UI for example will be pulling tasks instead of execution trying to populate old runs input/output data
* We can make it so that if it's not present then it looks at mlmd executions
* If MLMD executions are not available we don't render the graphs? 
The problem exists even if we use a new table though.

User must opt in to migrating via configs.

We can either: 
1. User needs to provision (as new configs) DB credentials to apiserver, if the user does not provide these - api server will fail to start up with reason
2. Or we auto detect if we can see these tables (if they are using the opinioated kf/kfp installs), if so we require an opt-in config via `MigrateMLMD=True`
   This way the user does not need to provide an additional set of configs. If we can't access it, user will need to provide credentials like (1)
3. Migration script - light weight, it's a one time op, API Server would fail if we are in a pre-migrate state 

### Testing

[//]: # (TODO: add)

### Conclusion


