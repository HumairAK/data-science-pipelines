-- This is a static table to keep type values normalized
-- It will contain types like: Artifact, Model, Dataset
CREATE TABLE `artifact_type`
(
    `UUID` int NOT NULL AUTO_INCREMENT,
    `name` varchar(255) NOT NULL,
    `version` varchar(255) DEFAULT NULL,
    `description` text,
    PRIMARY KEY (`UUID`),
    KEY `idx_type_name` (`name`)
);

CREATE TABLE `artifacts`
(
    `UUID` varchar(191) NOT NULL,
    -- For multi-tenancy, this is new and did not exist in mlmd, for migration we can fetch the associated context and backfill this
    `Namespace` varchar(63) NOT NULL,
    `type_id` int NOT NULL,
    `uri` text,
    `name` varchar(128) DEFAULT NULL,
    `create_time_since_epoch` bigint NOT NULL DEFAULT '0',
    `last_update_time_since_epoch` bigint NOT NULL DEFAULT '0',
    CONSTRAINT fk_artifacts_types FOREIGN KEY (type_id) REFERENCES artifact_type (UUID) ON DELETE RESTRICT ON UPDATE CASCADE,
    KEY idx_artifacts_type_id (type_id)
);

-- For supporting Artifact metadata
CREATE TABLE `artifact_properties`
(
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

-- Analogous to an mlmd Event, except it is specific to artifacts <-> tasks (instead of executions)
CREATE TABLE `artifact_events`
(
    `id` int NOT NULL AUTO_INCREMENT,
    `artifact_id` int NOT NULL,
    `task_id` int NOT NULL,
    -- 0 for INPUT, 1 for OUTPUT
    -- Optionally, if we don't want a CHECK here, we can just do validation server side
    `type` int NOT NULL CHECK (type IN (0, 1)),
    `create_time_since_epoch` bigint NOT NULL DEFAULT '0',

    UNIQUE KEY `UniqueLink` (`artifact_id`,`task_id`,`type`),
    KEY `idx_link_task_id` (`task_id`),
    CONSTRAINT fk_artifact_events_artifacts FOREIGN KEY (artifact_id) REFERENCES artifacts (UUID) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_artifact_events_tasks FOREIGN KEY (task_id) REFERENCES tasks (UUID) ON DELETE CASCADE ON UPDATE CASCADE
);

-- This is a static table to keep type values normalized
-- TODO: do we need a "NestedPipeline" TaskType?
-- It will contain types like: Runtime, ConditionBranch, Condition, Loop, LoopIteration
-- Optionally, we can consolidate TaskType and ArtifactType into one table

-- The Types: `ConditionBranch`, `Condition`, `Loop` are taskgroups in the KFP SDK and each have a dag
-- driver execution associated with it, so these need to be converted to Tasks. `LoopIteration` is another dag
-- driver that is a child to "Loop" task type. The `Loop` task retains the IterationCount, and groups all iterations
-- withing a loop, whereas the `LoopIteration` tracks the IterationIndex of a given iteration in the loop. The
-- existing of `LoopIteration` is unfortunate, and the project should try to remove the need for this, however this
-- is out of scope of this proposal.

-- TODO: Get rid of this --> Have task type be a string enum
CREATE TABLE `task_types`
(
    `UUID` int NOT NULL AUTO_INCREMENT,
    `name` varchar(255) NOT NULL,
    `version` varchar(255) DEFAULT NULL,
    `description` text,
    PRIMARY KEY (`UUID`),
    KEY `idx_type_name` (`name`)
);

-- Consider breaking up tasks table into sub tables like task_iteration, task_condition, etc.
-- We can do inheritance to have the different task types,
-- which would mean having required fields in a shared task field, and then
-- separate table for differnt fields (e.g. task_iteration would have iteration_count
-- field and a task_uuid FK column).
CREATE TABLE `tasks`
(
    `UUID` varchar(255) NOT NULL,
    `Namespace` varchar(255) NOT NULL,
    `PipelineName` varchar(255) NOT NULL,
    `RunUUID` varchar(255) NOT NULL,
    -- Only applicable for Runtime Tasks. This doesn't point to the runtime pod for this task, this will need to be fixed
    `PodName` varchar(255) NOT NULL,
    -- This likely doesn't get used and should be dropped, it has no constraints
    -- TODO: why is execution id only set sometimes? for container executions? does driver do it?
    `MLMDExecutionID` varchar(255) NOT NULL,
    `CreatedTimestamp` bigint NOT NULL,
    `StartedTimestamp` bigint DEFAULT '0',
    `FinishedTimestamp` bigint DEFAULT '0',
    `Fingerprint` varchar(255) NOT NULL,
    -- This seems like it's unique within a run, regardless of task type
    `Name` varchar(255) DEFAULT NULL,
    `ParentTaskUUID` varchar(255) DEFAULT NULL,
    -- TODO: only set sometimes?
    `State` varchar(255) DEFAULT NULL,
    `StateHistory` longtext,
    -- MLMDInputs/MLMDOutputs are not used today and will be dropped
    `MLMDInputs` longtext,
    `MLMDOutputs` longtext,
    `ChildrenPods` longtext,
    `Payload` longtext,
    -- New fields:

    -- Only applicable for Runtime
    `DisplayName` varchar(255),
    `InputParameters` longtext,
    `OutputParameters` longtext,
    -- Only applicable for IterationIndex tasks
    `IterationIndex` int,
    -- Only applicable for Iteration tasks
    `IterationCount` int,
    -- Corresponds to the executions created for each driver pod, which result in a Node on the Run Graph.
    `TypeID` int NOT NULL,

    PRIMARY KEY (`UUID`),
    KEY `tasks_RunUUID_run_details_UUID_foreign` (`RunUUID`),
    CONSTRAINT fk_tasks_types FOREIGN KEY (TypeID) REFERENCES task_types (UUID) ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT `tasks_RunUUID_run_details_UUID_foreign` FOREIGN KEY (`RunUUID`) REFERENCES `run_details` (`UUID`) ON DELETE CASCADE ON UPDATE CASCADE,
    KEY idx_artifacts_type_id (TypeID)
)


-- The run_details table doesn't change but is provided here as a reference
CREATE TABLE `run_details`
(
    `UUID` varchar(255) NOT NULL,
    `DisplayName` varchar(255) NOT NULL,
    `Name` varchar(255) NOT NULL,
    `Description` varchar(255) NOT NULL,
    `Namespace` varchar(255) NOT NULL,
    `ExperimentUUID` varchar(255) NOT NULL,
    `JobUUID` varchar(255) DEFAULT NULL,
    `StorageState` varchar(255) NOT NULL,
    `ServiceAccount` varchar(255) NOT NULL,
    `PipelineId` varchar(255) NOT NULL,
    `PipelineVersionId` varchar(255) DEFAULT NULL,
    `PipelineName` varchar(255) NOT NULL,
    `PipelineSpecManifest` longtext,
    `WorkflowSpecManifest` longtext,
    `Parameters` longtext,
    `RuntimeParameters` longtext,
    `PipelineRoot` longtext,
    `CreatedAtInSec` bigint NOT NULL,
    `ScheduledAtInSec` bigint DEFAULT '0',
    `FinishedAtInSec` bigint DEFAULT '0',
    `Conditions` varchar(255) NOT NULL,
    `State` varchar(255) DEFAULT NULL,
    `StateHistory` longtext,
    `PipelineRuntimeManifest` longtext NOT NULL,
    `WorkflowRuntimeManifest` longtext NOT NULL,
    `PipelineContextId` bigint DEFAULT '0',
    `PipelineRunContextId` bigint DEFAULT '0',
    PRIMARY KEY (`UUID`),
    KEY `experimentuuid_createatinsec` (`ExperimentUUID`,`CreatedAtInSec`),
    KEY `experimentuuid_conditions_finishedatinsec` (`ExperimentUUID`,`Conditions`,`FinishedAtInSec`),
    KEY `namespace_createatinsec` (`Namespace`,`CreatedAtInSec`),
    KEY `namespace_conditions_finishedatinsec` (`Namespace`,`Conditions`,`FinishedAtInSec`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci