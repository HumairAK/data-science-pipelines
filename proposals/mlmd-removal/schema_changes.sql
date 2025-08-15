CREATE TABLE `artifacts`
(
    `UUID` varchar(191) NOT NULL,
    `namespace` varchar(63) NOT NULL, -- enables multi-tenancy on artifacts
    `type` varchar(64) DEFAULT NULL, -- examples: Artifact, Model, Dataset
    `uri` text,
    `name` varchar(128) DEFAULT NULL,
    `create_time_since_epoch` bigint NOT NULL DEFAULT '0',
    `last_update_time_since_epoch` bigint NOT NULL DEFAULT '0',
    `properties` JSON DEFAULT NULL, -- equivalent to mlmd custom properties
    PRIMARY KEY (`UUID`)
);

-- Analogous to an mlmd Event, except it is specific to artifacts <-> tasks (instead of executions)
CREATE TABLE `artifact_events`
(
    `UUID` varchar(191) NOT NULL,
    `artifact_id` varchar(191) NOT NULL,
    `task_id` varchar(191) NOT NULL,
    -- 0 for INPUT, 1 for OUTPUT
    `type` int NOT NULL,
    `create_time_since_epoch` bigint NOT NULL DEFAULT '0',

    PRIMARY KEY (`UUID`),
    UNIQUE KEY `UniqueLink` (`artifact_id`,`task_id`,`type`),
    KEY `idx_link_task_id` (`task_id`),
    KEY `idx_link_artifact_id` (`artifact_id`),

    CONSTRAINT fk_artifact_events_tasks FOREIGN KEY (task_id) REFERENCES tasks (UUID) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_artifact_events_artifacts FOREIGN KEY (artifact_id) REFERENCES artifacts (UUID) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE `tasks`
(
    `UUID` varchar(191) NOT NULL,
    `Namespace` varchar(63) NOT NULL, -- updated to 63 (max namespace size in k8s)
    -- This is used for searching for cached_fingerprints today
    -- likely to prevent caching across pipelines 
    `PipelineName` varchar(128) NOT NULL,
    `RunUUID` varchar(191) NOT NULL,
    `PodName` varchar(255) NOT NULL, -- This is broken today and will need to be fixed
    `CreatedTimestamp` bigint NOT NULL,
    `StartedTimestamp` bigint DEFAULT '0',
    `FinishedTimestamp` bigint DEFAULT '0',
    `Fingerprint` varchar(255) NOT NULL,
    `Name` varchar(128) DEFAULT NULL,
    `ParentTaskUUID` varchar(191) DEFAULT NULL,
    `State` varchar(64) DEFAULT NULL,
    `StateHistory` longtext,
    -- Remove the following: 
    -- `MLMDExecutionID` varchar(255) NOT NULL,
    -- `MLMDInputs` longtext,
    -- `MLMDOutputs` longtext,
    -- `ChildrenPods` longtext,
    -- `Payload` longtext,

    -- New fields:
    `InputParameters` longtext,
    `OutputParameters` longtext,
    -- Corresponds to the executions created for each driver pod, which result in a Node on the Run Graph.
    -- E.g values are: Runtime, Condition, Loop, etc.
    `Type` varchar(64) NOT NULL,
    -- All type-specific attributes (Runtime.DisplayName, Loop.IterationIndex/Count)
    `TypeAttrs` json NOT NULL,

    PRIMARY KEY (`UUID`),
    KEY idx_artifacts_type_id (Type),
    KEY idx_pipeline_name (PipelineName),
    KEY idx_parent_run (`RunUUID`, `ParentTaskUUID`),
    KEY idx_parent_task_uuid (ParentTaskUUID),
    CONSTRAINT `tasks_RunUUID_run_details_UUID_foreign` FOREIGN KEY (`RunUUID`) REFERENCES `run_details` (`UUID`) ON DELETE CASCADE ON UPDATE CASCADE,
)

-- We will also revamp the Metrics table, it is not used today so we can drop it 
-- and recreate it as needed without worrying about breaking changes
CREATE TABLE `run_metrics`
(
    `TaskUUID` varchar(191) NOT NULL,
    `Name` varchar(128) NOT NULL,
    `NumberValue` double DEFAULT NULL,
    `Namespace` varchar(63) NOT NULL,
    `JsonValue` JSON DEFAULT NULL,
    -- 0 for INPUT, 1 for OUTPUT
    `Type` int NOT NULL,
    `Schema` varchar(64) NOT NULL,
    
    PRIMARY KEY (`TaskUUID`, `Name`),
    KEY idx_number_value (NumberValue)
    CONSTRAINT fk_run_metrics_tasks FOREIGN KEY (TaskUUID) REFERENCES tasks (UUID) ON DELETE CASCADE ON UPDATE CASCADE
)
