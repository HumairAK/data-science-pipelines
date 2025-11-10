# PipelineTaskDetailTaskType

 - ROOT: Root task replaces Root Execution, it is the top ancestor task to all tasks in the pipeline run  - CONDITION_BRANCH: Condition Branch is the wrapper If block  - CONDITION: Condition is an individual if branch (this feels counter intuitive but this is how it's named in the SDK IR) and we are consistent with the naming here.  - LOOP: Task Group for Condition Branches Task Group for Loop Iterations  - DAG: Generic DAG task type for types like Nested Pipelines where there is no declarative way to detect this within a driver.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


