# Kubeflow Pipelines ML-Metadata (MLMD) Removal Proposal

## Proposed Solution

[//]: # (TODO: add)

### New KFP Database Schema

We will make the following Additions to the DB Schema for Artifacts support: 

```mysql
# This is a static table to keep type values normalized 
# It will contain types like: Artifact, Model, Dataset 
CREATE TABLE `artifact_type` (
    `UUID` int NOT NULL AUTO_INCREMENT,
    `name` varchar(255) NOT NULL,
    `version` varchar(255) DEFAULT NULL,
    `description` text,
    PRIMARY KEY (`UUID`),
    KEY `idx_type_name` (`name`)
);

CREATE TABLE `artifacts` (
   `UUID` varchar(191) NOT NULL,
   `Namespace` varchar(63) NOT NULL, # For multi-tenancy, this is new and did not exist in mlmd, for migration we can fetch the associated context and backfill this
   `type_id` int NOT NULL, 
   `uri` text,
   `name` varchar(128) DEFAULT NULL,
   `create_time_since_epoch` bigint NOT NULL DEFAULT '0',
   `last_update_time_since_epoch` bigint NOT NULL DEFAULT '0',
   CONSTRAINT fk_artifacts_types FOREIGN KEY (type_id) REFERENCES artifact_type (UUID) ON DELETE RESTRICT ON UPDATE CASCADE,
   KEY idx_artifacts_type_id (type_id)
);

# For supporting Artifact metadata
CREATE TABLE `artifact_properties` (
   `artifact_id` integer,
   `name` varchar(128) NOT NULL,
   `is_custom_property` bool,
   `int_value` int,
   `double_value` double,
   `string_value` varchar(128),
   `byte_value` mediumblob,
   `proto_value` mediumblob,
   `bool_value` bool,
   CONSTRAINT fk_artifact_properties_artifacts FOREIGN KEY (artifact_id) REFERENCES artifacts (UUID) ON DELETE CASCADE ON UPDATE CASCADE,
   PRIMARY KEY (`artifact_id`, `name`)
);

# Analogous to an mlmd Event, except it is specific to artifacts <-> tasks (instead of executions)
CREATE TABLE `artifact_events` (
   `id` int NOT NULL AUTO_INCREMENT,
   `artifact_id` int NOT NULL,
   `task_id` int NOT NULL,
   # 0 for INPUT, 1 for OUTPUT
   # Optionally, if we don't want a CHECK here, we can just do validation server side
   `type` int NOT NULL CHECK (type IN (0, 1)),
   `create_time_since_epoch` bigint NOT NULL DEFAULT '0',
   
   PRIMARY KEY (`id`),
   UNIQUE KEY `UniqueLink` (`artifact_id`,`task_id`,`type`),
   KEY `idx_link_task_id` (`task_id`),
   CONSTRAINT fk_artifact_events_artifacts FOREIGN KEY (artifact_id) REFERENCES artifacts (UUID) ON DELETE CASCADE ON UPDATE CASCADE
);
```

The following changes and additions will be made to task tables (some pre-existing fields are omitted for clarity) related to tasks:
Note that a task is a db model for a task node type as viewed in the Run Graph of the UI.

```mysql

# This is a static table to keep type values normalized 
# It will contain types like: Runtime, ConditionBranch, Condition, Loop, LoopIteration
#  Optionally, we can consolidate TaskType and ArtifactType into one table 
CREATE TABLE `TaskType` (
    `UUID` int NOT NULL AUTO_INCREMENT,
    `name` varchar(255) NOT NULL,
    `version` varchar(255) DEFAULT NULL,
    `description` text,
    PRIMARY KEY (`UUID`),
    KEY `idx_type_name` (`name`)
);

CREATE TABLE `tasks` (
  `UUID` varchar(255) NOT NULL,
  `Name` varchar(255) DEFAULT NULL,             # This seems like it's unique within a run, regardless of task type
  `RunUUID` varchar(255) NOT NULL,
  `PodName` varchar(255) NOT NULL,              # Only applicable for Runtime Tasks. This doesn't point to the runtime pod for this task, this will need to be fixed
  `MLMDExecutionID` varchar(255) NOT NULL,      # This likely doesn't get used and should be dropped, it has no constraints
                                                # TODO: why is execution id only set sometimes? for container executions? does driver do it?  
  `Fingerprint` varchar(255) NOT NULL,
  `ParentTaskUUID` varchar(255) DEFAULT NULL,
  `State` varchar(255) DEFAULT NULL,            # TODO: only set sometimes?
  `MLMDInputs` longtext,                        # These are not used today and will be dropped
  `MLMDOutputs` longtext,                       # These are not used today and will be dropped
  `Payload` longtext,
   # New fields:
  `DisplayName` varchar(255),                   # Only applicable for Runtime 
  `InputParameters` longtext,
  `OutputParameters` longtext,
  `IterationIndex` int,                         # Only applicable for IterationIndex tasks 
  `IterationCount` int,                         # Only applicable for Iteration tasks 
  `TypeID` int NOT NULL,
  CONSTRAINT fk_tasks_types FOREIGN KEY (TypeID) REFERENCES TaskType (UUID) ON DELETE RESTRICT ON UPDATE CASCADE,
  KEY idx_artifacts_type_id (TypeID)
)
```

Note we can also do inheritance to have the different task types, which would mean having required fields in a shared task field, and then separate table for differnt fields (e.g. task_iteration would have iteration_count field and a task_uuid FK column).

The `Type` Column corresponds to the executions created for each driver pod, which result in a Node on the Run Graph. 

The following Types: `ConditionBranch`, `Condition`, `Loop` are taskgroups in the KFP SDK and each have a dag driver execution associated with it, so these need to be converted to Tasks. `LoopIteration` is another dag driver that is a child to "Loop" task type. The `Loop` task retains the IterationCount, and groups all iterations withing a loop, whereas the `LoopIteration` tracks the IterationIndex of a given iteration in the loop. The existing of `LoopIteration` is unfortunate, and the project should try to remove the need for this, however this is out of scope of this proposal. 

### KFP Server API

To facilitate the removal of MLMD, the KFP server will now take on the burden of handling Artifacts, Dags, input resolution, and so on. 

Instead of an MLMD client, the Driver and Launcher will include and leverage a `v2beta1.RunServiceClient` retrieved via `NewRunServiceClient()` in `backend/api/v2beta1`.

The Backend API protos will require the following new message types: 

```protobuf
syntax = "proto3";

// Note to be confused with RuntimeArtifact in pipelinespec
message Artifact {
  // Output only. The unique server generated id of the artifact.
  // Note: Updated id name to be consistent with other api naming patterns (with prefix)
  int64 artifact_id = 1;
  // The client provided name of the artifact.
  // Note: it seems in MLMD when name was set, it had to be unique for that type_id
  // this restriction is removed here
  string name = 2;
  // The id of an ArtifactType. This needs to be specified when an artifact is
  // created, and it cannot be changed.
  int64 type_id = 3;
  // Output only. The name of an ArtifactType. E.g. Dataset
  string type = 4;
  // The uniform resource identifier of the physical artifact.
  // May be empty if there is no physical artifact.
  string uri = 5;
  // User provided custom properties which are not defined by its type.
  map<string, Value> custom_properties = 6;
  // Output only. Create time of the artifact in millisecond since epoch.
  // Note: The type and name is updated from mlmd artifact to be consistent with other backend apis.
  google.protobuf.Timestamp created_at = 7;
  
  // New field: 
  string namespace = 8;
  
  // In KFP only the Live state is ever used
  // Optionally we can omit this and always assume 
  // an artifact fetched is "Live" and add a state 
  // later if needed 
  enum State {
    UNKNOWN = 0;
    PENDING = 1;
    LIVE = 2;
    // omitted other states as they are not applicable to KFP
  }
  // The state of the artifact known to the system.
  State state = 6;

  // Fields not included from mlmd artifact are: state, external_id, properties, system_metadata, last_update_time_since_epoch
  // Reference: https://raw.githubusercontent.com/kubeflow/pipelines/refs/heads/master/third_party/ml-metadata/ml_metadata/proto/metadata_store.proto
}
```

The Backend API protos will require the following new service endpoints:

```protobuf
syntax = "proto3";
service ArtifactService {
  // Finds all artifacts within the specified namespace.
  // Namespace field is required. In multi-user mode, the caller
  // is required to have RBAC verb "list" on the "artifacts"
  // resource for the specified namespace.
  rpc ListArtifacts(ListArtifactRequest) returns (ListArtifactResponse) {
    option (google.api.http) = {
      get: "/apis/v2beta1/artifacts"
    };
  }

  // Finds a specific Artifact by ID.
  rpc GetArtifact(GetArtifactRequest) returns (Artifact) {
    option (google.api.http) = {
      get: "/apis/v2beta1/artifacts/{artifact_id}"
    };
  }
  
}

message GetArtifactRequest {
  // Required. The ID of the artifact to be retrieved.
  string artifact_id = 1;
}

// Note: This follows the same format as other List operations in KFP backend
message ListArtifactRequest {
  // Optional input. Namespace for the artifacts.
  string namespace = 1;

  // A page token to request the results page.
  string page_token = 2;

  // The number of artifacts to be listed per page. If there are more artifacts
  // than this number, the response message will contain a valid value in the
  // nextPageToken field.
  int32 page_size = 3;

  // Sorting order in form of "field_name", "field_name asc" or "field_name desc".
  // Ascending by default.
  string sort_by = 4;

  // A url-encoded, JSON-serialized filter protocol buffer (see
  // [filter.proto](https://github.com/kubeflow/artifacts/blob/master/backend/api/filter.proto)).
  string filter = 5;
}
```

To run.proto we will add: 

```protobuf
syntax = "proto3";

service RunService {
  rpc ListArtifactEvents(GetArtifactEventsRequest) returns (GetArtifactEventsResponse) {
    option (google.api.http) = {
      get: "/apis/v2beta1/artifafct_events"
    };
  }
}

// The fields here work the same as previous backend api calls 
message GetArtifactEventsRequest {
  // Filter event by a set of task_ids 
  repeated string task_ids = 1;
  string page_token = 2;
  int32 page_size = 3;
  string sort_by = 4;
  string filter = 5;
}

message GetArtifactEventsResponse {
  repeated ArtifactEvents events = 1;
  int32 total_size = 2;
  string next_page_token = 3;
}

message ArtifactEvents {
  int64 id = 1;
  int64 artifact_id = 2;
  int64 task_id = 3;
  enum Type {
    INPUT = 0;
    OUTPUT = 1;
  }
  Type type = 4;
  google.protobuf.Timestamp created_at = 5;
}

// TODO: getArtifactEventsByTasks
message PipelineTaskDetail {
  // omit pre-existing fields 
  
  repeated string child_tasks_ids = 17;
  repeated Artifact input_artifacts = 18; 
  repeated Artifact output_artifacts = 19;

  message parameter {
    string key = 1;
    string value = 2;
  }
  repeated parameter input_parameters = 20;
  repeated parameter output_parameters = 21;

  enum TaskType { 
    RUNTIME = 0;
    CONDITION_BRANCH = 1; 
    CONDITION = 2; 
    LOOP = 3; 
    LOOP_ITERATION = 4;
  }
  TaskType type = 22;
  
  // Applies to type LOOP_ITERATION
  int64 iteration_index = 23; 
  // Applies to type LOOP
  int64 iteration_count = 24; 
  string name = 25;
  
  // Deprecate these fields 
  map<string, ArtifactList> inputs = 11;
  map<string, ArtifactList> outputs = 12;
  repeated ChildTask child_tasks = 16;
}
```

Example Run JSON would look like the following:

```json
{
  "experiment_id": "f7344db6-de5d-4e68-816b-98b4f0d1ca7f",
  "run_id": "9e68aca5-3afa-4028-8777-f697d858053f",
  "display_name": "mypipeline-run",
  "storage_state": "AVAILABLE",
  "pipeline_version_reference": {},
  "pipeline_root": "minio://mlpipeline/v2/artifacts/pipeline/f7344db6-de5d-4e68-816b-98b4f0d1ca7f",
  "runtime_config": {},
  "service_account": "pipeline-runner",
  "created_at": "2025-08-08T15:15:41Z",
  "scheduled_at": "2025-08-08T15:15:41Z",
  "finished_at": "2025-08-08T15:18:24Z",
  "state": "SUCCEEDED",
  "run_details": {
    "task_details": [
      {
        "run_id": "run_id",
        "task_id": "task_id_1",
        "display_name": "train-model",
        "create_time": "2025-08-08T15:15:41Z",
        "start_time": "2025-08-08T15:17:36Z",
        "end_time": "2025-08-08T15:18:24Z",
        "state": "SUCCEEDED",
        "state_history": [],
        "child_tasks": [],
        // deprecated
        // New fields
        "type": "RUNTIME",
        // This should match the component task names created during sdk compilation
        // UI can use this to look up matching tasks in the UI.
        // In the case of task_groups this would take on names like: "condition-branches-1, for-loop-1, etc."
        "name": "task_name",
        "input_parameters": [
          {
            "min_max_scaler": "false"
          },
          {
            // When the parameter is from an upstream task, this is how it's stored
            // in mlmd executions today.
            // Ideallly the parameter struct would be a bit more informative
            // I.e. have fields like "ProducerTask", "Type" (runtime, task_output, etc.)
            // but this best left as a follow up improvement and out of scope
            "pipelinechannel--output-msg-a_msg": "this"
          }
        ],
        "output_parameters": [
          {
            "my_output": "some_value"
          }
        ],
        "input_artifacts": [
          {
            "name": "input_dataset",
            "uri": "minio://mlpipeline/v2/artifacts/pipeline/9e68aca5-3afa-4028-8777-f697d858053f/input_dataset",
            "artifact_id": "5",
            "type": "Model",
            "namespace": "some_namespace",
            "created_at": "2025-08-08T15:15:41Z"
          }
        ],
        // same structure as Input_artifacts
        "output_artifacts": [],
        // Included for LoopIteration
        "iteration_index": 2,
        // Included for LoopCount, iteration_index & iteration_count are mutually exclusive
        "iteration_count": 2,
        "child_task_ids": [
          "task_id_2",
          "task_id_3"
        ]
      }
    ]
  },
  "state_history": []
}
```

### Driver changes

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

##### ContainerExecution

[//]: # (TODO: Add the standard code th[//]: # (TODO: Add the standard code that will need to be changed for Container executions, talk about input resolutions, how will input parameters be handled?)
at will need to be changed for Container executions, talk about input resolutions, how will input parameters be handled?)

##### Loops 

Loops in KFP today require two types of dags, there's either a dag that has an `iteration_count` or a dag that has an `iteration_index`. 
We'll refer to these as Loop and LoopIteration respectively. A `Loop` is a task grouping of components that will run within a loop and trackts the total count of iterations via `iteration_count`. A `LoopIteration` is a dag that tracks the current iteration for a given loop via `iteration_index`.

Each of these results in an `DagExecution`. The components that run for each iteration will also have their regular `ContainerExecutions`.
Each of these are used to resolve inputs/outputs and will need to be logged as Tasks into the Tasks table. 

[//]: # (TODO: Add the code that will need to be replaced)

##### Exit handler 

##### DSL If/Else/ElseIf

When working with Conditions in KFP, new nodes are introduced in the Pipeline Graph, they are prefixed with `condition-` 
or `condition-branches-`. 

1. **`CONDITION`** - Represents a conditional task group (If/Else/ElseIf), in KFP it is represented by a DAG driver that outputs a condition parameter which determines whether the underlying dag or components should execute.

2. **`CONDITION_BRANCHES`** - Represents the branches that stem from a conditional statement.

### Launcher changes 

* Status updates and reporting for task level

### Nested Pipelines 

[//]: # (TODO: add)

### Caching 


[//]: # (TODO: add)

### Frontend Changes 


For run details, MLMD data is fetched via: 

```typescript
const context = await getKfpV2RunContext(runId);
const executions = await getExecutionsFromContext(context);
const artifacts = await getArtifactsFromContext(context);
const events = await getEventsByExecutions(executions);
```

These can be replaced by: 

```typescript
// context no longer needed, use the Run object which often readily available wherever context is required
const tasks = run.run_details.task_details;  // run is a V2beta1Run
const artifacts = await fetchArtifactsFromTasks(tasks); // the information is now available in the task.
// a separate call for this is may not needed, as the required info may already be present in `tasks`
const events = await getArtifactEventsByTasks(tasks); // uses ListArtifactEvents()
```


[//]: # (TODO: add)

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

[//]: # (TODO: add)

### Testing

[//]: # (TODO: add)

### Conclusion


