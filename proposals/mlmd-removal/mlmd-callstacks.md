root/dag/container.go

- mlmd.GetPipeline(ctx, opts.PipelineName, opts.RunID,..)
- mlmd.GetDAG(ctx, opts.DAGExecutionID)
- mlmd.CreateExecution(ctx, pipeline, ecfg)
- mlmd.PublishExecution(ctx, createdExecution, executorOutput.GetParameterValues(), outputArtifacts, pb.Execution_CACHED)
- reuseCachedOutputs(ctx, ExecutorInput, mlmd, CachedMLMDExecutionID)
- resolveInputs(ctx, dag, iterationIndex, pipeline, opts, mlmd, expr)
    - mlmd.GetInputArtifactsByExecutionID(ctx, dag.Execution.GetID()) -> inputs[artifact-name-from-path] -> RuntimeArtifactList(the list seems to only contain one artifact though)
        - replace with getting input artifacts for a task id
    - resolveInputParameter(ctx, dag, pipeline, opts, mlmd, toleration.GetTolerationJson(), inputParams)
        - resolveUpstreamParameters(cfg)
            - getDAGTasks(cfg.ctx, cfg.dag, cfg.pipeline, cfg.mlmd, nil)
            - mlmd.GetExecution(cfg.ctx, currentTask.GetExecution().GetCustomProperties()["parent_dag_id"].GetIntValue())
                - c.GetExecutions...
                - c.GetPipelineFromExecution(execution id)
                    - various mlmd calls to get Pipeline and PipelineRunCtx by the Execution ID
            - CollectInputs
                - collectContainerOutput
                    - cfg.mlmd.GetOutputArtifactsByExecutionId...
        - getDAGTasks(ctx, dag, pipeline, mlmd, nil)
            - mlmd.GetExecutionsInDAG(ctx, dag, pipeline, true)
                - Get all executions in a DAG, including subdags, the tasknames should apply GetTaskNameWithDagID() and GetParallelForTaskName(name, iteration_index) (if inside an iteration). Return a map of executions[tasknameSTR]->execution.
                - only mlmd call used is GetExecutionsByContext()
            - mlmd.GetDAG(ctx, v.GetExecution().GetId()).
                - c.GetExecutions(execution_id) -> returns Execution{execution, pipeline}
                    - users c.svc.GetExecutionsByID(ctx, req)
            - Recursive: getDAGTasks(ctx, subDAG, pipeline, mlmd, flattenedTasks)..
    - resolveInputArtifact()
        - resolveUpstreamArtifacts
            - getDAGTasks..
            - GetExecution..
            - GetOutputArtifactsByExecutionId(ctx, executionId) -> []OutputArtifact (which is a {name, mlmd.Artifact.pb, schema})
                - c.svc.GetEventsByExecutionIDs() (only pass the one id)
                - c.GetArtifacts, which calls c.svc.GetArtifactsByID(ctx, ArtifactIds) - note its get multiple artifacts

k8s.go
- kubernetesPlatformOps(ctx, mlmd, cacheClient, execution, ecfg, &opts) {}
    - publishDriverExecution(k8sClient, mlmd, ctx, createdExecution, outputParameters, nil, status)
        - mlmd.PrePublishExecution(ctx, execution, ecfg)
        - mlmd.PublishExecution...
    - createPVC(ctx, k8sClient, *execution, opts, cacheClient, mlmd, ecfg)
        - reuseCachedOutputs...
    - deletePVC(ctx, k8sClient, *execution, opts, cacheClient, mlmd, ecfg)
        - reuseCachedOutputs...
- extendPodSpecPatch(ctx, podSpec, opts, dag, pipeline, mlmd, inputParams)
    - resolveInputParameterStr(ctx, dag, pipeline, opts, mlmd, pvcNameParameter, inputParams)
        - resolveInputParameter...
    - resolveK8sJsonParameter(ctx, opts, dag, pipeline, mlmd, kubernetesExecutorConfig.GetNodeSelector().GetNodeSelectorJson(), inputParams, &nodeSelector)
        - resolveInputParameter...

cache.go
- reuseCachedOutputs(ctx, execution.ExecutorInput, mlmd, ecfg.CachedMLMDExecutionID)
    - Used by Container.go and createPVC and deletePVC
    - collectOutputArtifactMetadataFromCache(ctx, executorInput, cachedMLMDExecutionIDInt64, mlmd)
        - mlmd.GetOutputArtifactsByExecutionId(ctx, cachedMLMDExecutionID)
            - c.svc.GetEventsByExecutionIDs
