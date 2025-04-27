package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/expression"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
)

func validateContainer(opts Options) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("invalid container driver args: %w", err)
		}
	}()
	if opts.Container == nil {
		return fmt.Errorf("container spec is required")
	}
	return validateNonRoot(opts)
}

// provisionOuutputs prepares output references that will get saved to MLMD.
func provisionOutputs(
	pipelineRoot,
	taskName string,
	outputsSpec *pipelinespec.ComponentOutputsSpec,
	outputURISalt string,
	publishOutput string,
) *pipelinespec.ExecutorInput_Outputs {
	outputs := &pipelinespec.ExecutorInput_Outputs{
		Artifacts:  make(map[string]*pipelinespec.ArtifactList),
		Parameters: make(map[string]*pipelinespec.ExecutorInput_OutputParameter),
		OutputFile: component.OutputMetadataFilepath,
	}
	artifacts := outputsSpec.GetArtifacts()

	// TODO: Check if there's a more idiomatic way to handle this.
	if publishOutput == "true" {
		// Add a placeholder for a log artifact that will be written to by the
		// subsequent executor.
		if artifacts == nil {
			artifacts = make(map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec)
		}
		artifacts["executor-logs"] = &pipelinespec.ComponentOutputsSpec_ArtifactSpec{
			ArtifactType: &pipelinespec.ArtifactTypeSchema{
				Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{
					SchemaTitle: "system.Artifact",
				},
			},
		}
	}

	for name, artifact := range artifacts {
		outputs.Artifacts[name] = &pipelinespec.ArtifactList{
			Artifacts: []*pipelinespec.RuntimeArtifact{
				{
					// Do not preserve the query string for output artifacts, as otherwise
					// they'd appear in file and artifact names.
					Uri:      metadata.GenerateOutputURI(pipelineRoot, []string{taskName, outputURISalt, name}, false),
					Type:     artifact.GetArtifactType(),
					Metadata: artifact.GetMetadata(),
				},
			},
		}
	}

	for name := range outputsSpec.GetParameters() {
		outputs.Parameters[name] = &pipelinespec.ExecutorInput_OutputParameter{
			OutputFile: fmt.Sprintf("/tmp/kfp/outputs/%s", name),
		}
	}

	return outputs
}

func Container(ctx context.Context, opts Options, mlmd *metadata.Client, cacheClient *cacheutils.Client) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.Container(%s) failed: %w", opts.info(), err)
		}
	}()
	b, _ := json.Marshal(opts)
	glog.V(4).Info("Container opts: ", string(b))
	err = validateContainer(opts)
	if err != nil {
		return nil, err
	}
	var iterationIndex *int
	if opts.IterationIndex >= 0 {
		index := opts.IterationIndex
		iterationIndex = &index
	}
	// TODO(Bobgy): there's no need to pass any parameters, because pipeline
	// and pipeline run context have been created by root DAG driver.
	pipeline, err := mlmd.GetPipeline(ctx, opts.PipelineName, opts.RunID, "", "", "", "")
	if err != nil {
		return nil, err
	}
	dag, err := mlmd.GetDAG(ctx, opts.DAGExecutionID)
	if err != nil {
		return nil, err
	}
	glog.Infof("parent DAG: %+v", dag.Execution)
	expr, err := expression.New()
	if err != nil {
		return nil, err
	}
	inputs, err := resolveInputs(ctx, dag, iterationIndex, pipeline, opts, mlmd, expr)
	if err != nil {
		return nil, err
	}

	executorInput := &pipelinespec.ExecutorInput{
		Inputs: inputs,
	}
	execution = &Execution{ExecutorInput: executorInput}
	condition := opts.Task.GetTriggerPolicy().GetCondition()
	if condition != "" {
		willTrigger, err := expr.Condition(executorInput, condition)
		if err != nil {
			return execution, err
		}
		execution.Condition = &willTrigger
	}

	// When the container image is a dummy image, there is no launcher for this
	// task. This happens when this task is created to implement a
	// Kubernetes-specific configuration, i.e., there is no user container to
	// run. It publishes execution details to mlmd in driver and takes care of
	// caching, which are usually done in launcher. We also skip creating the
	// podspecpatch in these cases.
	_, isKubernetesPlatformOp := dummyImages[opts.Container.Image]
	if isKubernetesPlatformOp {
		// To be consistent with other artifacts, the driver registers log
		// artifacts to MLMD and the launcher publishes them to the object
		// store. This pattern does not work for kubernetesPlatformOps because
		// they have no launcher. There's no point in registering logs that
		// won't be published. Consequently, when we know we're dealing with
		// kubernetesPlatformOps, we set publishLogs to "false". We can amend
		// this when we update the driver to publish logs directly.
		opts.PublishLogs = "false"
	}

	if execution.WillTrigger() {
		executorInput.Outputs = provisionOutputs(
			pipeline.GetPipelineRoot(),
			opts.Task.GetTaskInfo().GetName(),
			opts.Component.GetOutputDefinitions(),
			uuid.NewString(),
			opts.PublishLogs,
		)
	}

	ecfg, err := metadata.GenerateExecutionConfig(executorInput)
	if err != nil {
		return execution, err
	}
	ecfg.TaskName = opts.Task.GetTaskInfo().GetName()
	ecfg.ExecutionType = metadata.ContainerExecutionTypeName
	ecfg.ParentDagID = dag.Execution.GetID()
	ecfg.IterationIndex = iterationIndex
	ecfg.NotTriggered = !execution.WillTrigger()

	if isKubernetesPlatformOp {
		return execution, kubernetesPlatformOps(ctx, mlmd, cacheClient, execution, ecfg, &opts)
	}

	// Generate fingerprint and MLMD ID for cache
	fingerPrint, cachedMLMDExecutionID, err := getFingerPrintsAndID(execution, &opts, cacheClient)
	if err != nil {
		return execution, err
	}
	ecfg.CachedMLMDExecutionID = cachedMLMDExecutionID
	ecfg.FingerPrint = fingerPrint

	var createdExecution *metadata.Execution
	if opts.DevMode {
		createdExecution, err = mlmd.GetExecution(ctx, opts.DevExecutionId)
	} else {
		createdExecution, err = mlmd.CreateExecution(ctx, pipeline, ecfg)
	}
	if err != nil {
		return nil, err
	}

	executionID := createdExecution.GetID()
	parentID := &ecfg.ParentDagID
	// if this parent is an iteration, get the parent's parent (iterator) dag:
	// note an iterator dag spawns iteration dags
	//iterationValue, exists := dag.Execution.GetExecution().CustomProperties["iteration_index"]
	//isIteration := exists && iterationValue.GetIntValue() >= 0
	//if !isIteration {
	//	glog.Infof("parent is isIteration, get the parent's parent (iterator) dag")
	//	parentDagID := dag.Execution.GetExecution().CustomProperties["parent_dag_id"].GetIntValue()
	//	iteratorParent, err1 := mlmd.GetDAG(ctx, parentDagID)
	//	if err1 != nil {
	//		return nil, err1
	//	}
	//	t := iteratorParent.Execution.GetID()
	//	parentID = &t
	//	glog.Infof("got iteration parent dag id %d", parentID)
	//}

	_, err = opts.MetadataClient.CreatePipelineRun(
		ctx,
		ecfg.TaskName,
		opts.PipelineName,
		opts.Namespace,
		"run-resource",
		pipeline.GetPipelineRoot(),
		pipeline.GetStoreSessionInfo(),
		opts.ExperimentId,
		parentID,
		&executionID,
	)
	if err != nil {
		return nil, err
	}

	for key, value := range inputs.ParameterValues {
		err := opts.MetadataClient.LogParameter(ctx, opts.ExperimentId, executionID, key, value.GetStringValue())
		if err != nil {
			return nil, err
		}
	}

	glog.Infof("Created execution: %s", createdExecution)
	execution.ID = createdExecution.GetID()

	if !execution.WillTrigger() {
		return execution, nil
	}

	// Use cache and skip launcher if all contions met:
	// (1) Cache is enabled
	// (2) CachedMLMDExecutionID is non-empty, which means a cache entry exists
	cached := false
	execution.Cached = &cached
	if opts.Task.GetCachingOptions().GetEnableCache() && ecfg.CachedMLMDExecutionID != "" {
		executorOutput, outputArtifacts, err := reuseCachedOutputs(ctx, execution.ExecutorInput, mlmd, ecfg.CachedMLMDExecutionID)
		if err != nil {
			return execution, err
		}
		// TODO(Bobgy): upload output artifacts.
		// TODO(Bobgy): when adding artifacts, we will need execution.pipeline to be non-nil, because we need
		// to publish output artifacts to the context too.
		if err := mlmd.PublishExecution(ctx, createdExecution, executorOutput.GetParameterValues(), outputArtifacts, pb.Execution_CACHED); err != nil {
			return execution, fmt.Errorf("failed to publish cached execution: %w", err)
		}
		glog.Infof("Use cache for task %s", opts.Task.GetTaskInfo().GetName())
		*execution.Cached = true
		return execution, nil
	}

	podSpec, err := initPodSpecPatch(
		opts.Container,
		opts.Component,
		executorInput,
		execution.ID,
		opts.PipelineName,
		opts.RunID,
		opts.PipelineLogLevel,
		opts.PublishLogs,
		opts.DAGExecutionID,
	)
	if err != nil {
		return execution, err
	}
	if opts.KubernetesExecutorConfig != nil {
		inputParams, _, err := dag.Execution.GetParameters()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch input parameters from execution: %w", err)
		}
		err = extendPodSpecPatch(ctx, podSpec, opts, dag, pipeline, mlmd, inputParams)
		if err != nil {
			return execution, err
		}
	}
	podSpecPatchBytes, err := json.Marshal(podSpec)
	if err != nil {
		return execution, fmt.Errorf("JSON marshaling pod spec patch: %w", err)
	}
	execution.PodSpecPatch = string(podSpecPatchBytes)
	return execution, nil
}
