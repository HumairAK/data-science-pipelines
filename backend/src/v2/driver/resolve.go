package driver

import (
	"context"
	"encoding/json"
	"errors"
	"google.golang.org/protobuf/encoding/protojson"

	"fmt"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/expression"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

// resolveUpstreamOutputsConfig is just a config struct used to store the input
// parameters of the resolveUpstreamParameters and resolveUpstreamArtifacts
// functions.
type resolveUpstreamOutputsConfig struct {
	ctx          context.Context
	paramSpec    *pipelinespec.TaskInputsSpec_InputParameterSpec
	artifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec
	dag          *metadata.DAG
	pipeline     *metadata.Pipeline
	mlmd         *metadata.Client
	err          func(error) error
}

var ErrResolvedParameterNull = errors.New("the resolved input parameter is null")

// resolvePodSpecInputRuntimeParameter resolves runtime value that is intended to be
// utilized within the Pod Spec. parameterValue takes the form of:
// "{{$.inputs.parameters['pipelinechannel--someParameterName']}}"
//
// parameterValue is a runtime parameter value that has been resolved and included within
// the executor input. Since the pod spec patch cannot dynamically update the underlying
// container template's inputs in an Argo Workflow, this is a workaround for resolving
// such parameters.
//
// If parameter value is not a parameter channel, then a constant value is assumed and
// returned as is.
func resolvePodSpecInputRuntimeParameter(parameterValue string, executorInput *pipelinespec.ExecutorInput) (string, error) {
	if isInputParameterChannel(parameterValue) {
		inputImage, err := extractInputParameterFromChannel(parameterValue)
		if err != nil {
			return "", err
		}
		if val, ok := executorInput.Inputs.ParameterValues[inputImage]; ok {
			return val.GetStringValue(), nil
		} else {
			return "", fmt.Errorf("executorInput did not contain container Image input parameter")
		}
	}
	return parameterValue, nil
}

// resolveK8sJsonParameter resolves a k8s JSON and unmarshal it
// to the provided k8s resource.
//
// Parameters:
//   - pipelineInputParamSpec: An input parameter spec that resolve to a valid JSON
//   - inputParams: InputParams that contain resolution context for pipelineInputParamSpec
//   - res: The k8s resource to unmarshal the json to
func resolveK8sJsonParameter[k8sResource any](
	ctx context.Context,
	opts Options,
	dag *metadata.DAG,
	pipeline *metadata.Pipeline,
	mlmd *metadata.Client,
	pipelineInputParamSpec *pipelinespec.TaskInputsSpec_InputParameterSpec,
	inputParams map[string]*structpb.Value,
	res *k8sResource,
) error {
	resolvedParam, err := resolveInputParameter(ctx, dag, pipeline, opts, mlmd,
		pipelineInputParamSpec, inputParams)
	if err != nil {
		return fmt.Errorf("failed to resolve k8s parameter: %w", err)
	}
	paramJSON, err := resolvedParam.GetStructValue().MarshalJSON()
	if err != nil {
		return err
	}
	err = json.Unmarshal(paramJSON, &res)
	if err != nil {
		return fmt.Errorf("failed to unmarshal k8s Resource json "+
			"ensure that k8s Resource json correctly adheres to its respective k8s spec: %w", err)
	}
	return nil
}

func resolveInputs(
	ctx context.Context,
	dag *metadata.DAG,
	iterationIndex *int,
	pipeline *metadata.Pipeline,
	opts Options,
	mlmd *metadata.Client,
	expr *expression.Expr,
) (inputs *pipelinespec.ExecutorInput_Inputs, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to resolve inputs: %w", err)
		}
	}()

	task := opts.Task
	inputsSpec := opts.Component.GetInputDefinitions()

	glog.V(4).Infof("dag: %v", dag)
	glog.V(4).Infof("task: %v", task)
	inputParams, _, err := dag.Execution.GetParameters()
	if err != nil {
		return nil, err
	}
	inputArtifacts, err := mlmd.GetInputArtifactsByExecutionID(ctx, dag.Execution.GetID())
	if err != nil {
		return nil, err
	}
	glog.Infof("parent DAG input parameters: %+v, artifacts: %+v", inputParams, inputArtifacts)
	inputs = &pipelinespec.ExecutorInput_Inputs{
		ParameterValues: make(map[string]*structpb.Value),
		Artifacts:       make(map[string]*pipelinespec.ArtifactList),
	}
	isIterationDriver := iterationIndex != nil

	handleParameterExpressionSelector := func() error {
		for name, paramSpec := range task.GetInputs().GetParameters() {
			var selector string
			if selector = paramSpec.GetParameterExpressionSelector(); selector == "" {
				continue
			}
			wrap := func(e error) error {
				return fmt.Errorf("resolving parameter %q: evaluation of parameter expression selector %q failed: %w", name, selector, e)
			}
			value, ok := inputs.ParameterValues[name]
			if !ok {
				return wrap(fmt.Errorf("value not found in inputs"))
			}
			selected, err := expr.Select(value, selector)
			if err != nil {
				return wrap(err)
			}
			inputs.ParameterValues[name] = selected
		}
		return nil
	}
	handleParamTypeValidationAndConversion := func() error {
		// TODO(Bobgy): verify whether there are inputs not in the inputs spec.
		for name, spec := range inputsSpec.GetParameters() {
			if task.GetParameterIterator() != nil {
				if !isIterationDriver && task.GetParameterIterator().GetItemInput() == name {
					// It's expected that an iterator does not have iteration item input parameter,
					// because only iterations get the item input parameter.
					continue
				}
				if isIterationDriver && task.GetParameterIterator().GetItems().GetInputParameter() == name {
					// It's expected that an iteration does not have iteration items input parameter,
					// because only the iterator has it.
					continue
				}
			}
			value, hasValue := inputs.GetParameterValues()[name]

			// Handle when parameter does not have input value
			if !hasValue && !inputsSpec.GetParameters()[name].GetIsOptional() {
				// When parameter is not optional and there is no input value, first check if there is a default value,
				// if there is a default value, use it as the value of the parameter.
				// if there is no default value, report error.
				if inputsSpec.GetParameters()[name].GetDefaultValue() == nil {
					return fmt.Errorf("neither value nor default value provided for non-optional parameter %q", name)
				}
			} else if !hasValue && inputsSpec.GetParameters()[name].GetIsOptional() {
				// When parameter is optional and there is no input value, value comes from default value.
				// But we don't pass the default value here. They are resolved internally within the component.
				// Note: in the past the backend passed the default values into the component. This is a behavior change.
				// See discussion: https://github.com/kubeflow/pipelines/pull/8765#discussion_r1119477085
				continue
			}

			switch spec.GetParameterType() {
			case pipelinespec.ParameterType_STRING:
				_, isValueString := value.GetKind().(*structpb.Value_StringValue)
				if !isValueString {
					// TODO(Bobgy): discuss whether we want to allow auto type conversion
					// all parameter types can be consumed as JSON string
					text, err := metadata.PbValueToText(value)
					if err != nil {
						return fmt.Errorf("converting input parameter %q to string: %w", name, err)
					}
					inputs.GetParameterValues()[name] = structpb.NewStringValue(text)
				}
			default:
				typeMismatch := func(actual string) error {
					return fmt.Errorf("input parameter %q type mismatch: expect %s, got %s", name, spec.GetParameterType(), actual)
				}
				switch v := value.GetKind().(type) {
				case *structpb.Value_NullValue:
					return fmt.Errorf("got null for input parameter %q", name)
				case *structpb.Value_StringValue:
					// TODO(Bobgy): consider whether we support parsing string as JSON for any other types.
					if spec.GetParameterType() != pipelinespec.ParameterType_STRING {
						return typeMismatch("string")
					}
				case *structpb.Value_NumberValue:
					if spec.GetParameterType() != pipelinespec.ParameterType_NUMBER_DOUBLE && spec.GetParameterType() != pipelinespec.ParameterType_NUMBER_INTEGER {
						return typeMismatch("number")
					}
				case *structpb.Value_BoolValue:
					if spec.GetParameterType() != pipelinespec.ParameterType_BOOLEAN {
						return typeMismatch("bool")
					}
				case *structpb.Value_ListValue:
					if spec.GetParameterType() != pipelinespec.ParameterType_LIST {
						return typeMismatch("list")
					}
				case *structpb.Value_StructValue:
					if spec.GetParameterType() != pipelinespec.ParameterType_STRUCT {
						return typeMismatch("struct")
					}
				default:
					return fmt.Errorf("parameter %s has unknown protobuf.Value type: %T", name, v)
				}
			}
		}
		return nil
	}
	// this function has many branches, so it's hard to add more postprocess steps
	// TODO(Bobgy): consider splitting this function into several sub functions
	defer func() {
		if err == nil {
			err = handleParameterExpressionSelector()
		}
		if err == nil {
			err = handleParamTypeValidationAndConversion()
		}
	}()
	// resolve input parameters
	if isIterationDriver {
		// resolve inputs for iteration driver is very different
		artifacts, err := mlmd.GetInputArtifactsByExecutionID(ctx, dag.Execution.GetID())
		if err != nil {
			return nil, err
		}
		inputs.ParameterValues = inputParams
		inputs.Artifacts = artifacts
		switch {
		case task.GetArtifactIterator() != nil:
			return nil, fmt.Errorf("artifact iterator not implemented yet")
		case task.GetParameterIterator() != nil:
			var itemsInput string
			if task.GetParameterIterator().GetItems().GetInputParameter() != "" {
				// input comes from outside the component
				itemsInput = task.GetParameterIterator().GetItems().GetInputParameter()
			} else if task.GetParameterIterator().GetItemInput() != "" {
				// input comes from static input
				itemsInput = task.GetParameterIterator().GetItemInput()
			} else {
				return nil, fmt.Errorf("cannot retrieve parameter iterator")
			}
			items, err := getItems(inputs.ParameterValues[itemsInput])
			if err != nil {
				return nil, err
			}
			if *iterationIndex >= len(items) {
				return nil, fmt.Errorf("bug: %v items found, but getting index %v", len(items), *iterationIndex)
			}
			delete(inputs.ParameterValues, itemsInput)
			inputs.ParameterValues[task.GetParameterIterator().GetItemInput()] = items[*iterationIndex]
		default:
			return nil, fmt.Errorf("bug: iteration_index>=0, but task iterator is empty")
		}
		return inputs, nil
	}

	// Handle parameters.
	for name, paramSpec := range task.GetInputs().GetParameters() {
		v, err := resolveInputParameter(ctx, dag, pipeline, opts, mlmd, paramSpec, inputParams)
		if err != nil {
			if !errors.Is(err, ErrResolvedParameterNull) {
				return nil, err
			}

			componentParam, ok := opts.Component.GetInputDefinitions().GetParameters()[name]
			if ok && componentParam != nil && componentParam.IsOptional {
				// If the resolved paramter was null and the component input parameter is optional, just skip setting
				// it and the launcher will handle defaults.
				continue
			}

			return nil, err
		}

		inputs.ParameterValues[name] = v
	}

	// Handle artifacts.
	for name, artifactSpec := range task.GetInputs().GetArtifacts() {
		v, err := resolveInputArtifact(ctx, dag, pipeline, mlmd, name, artifactSpec, inputArtifacts, task)
		if err != nil {
			return nil, err
		}
		inputs.Artifacts[name] = v
	}
	// TODO(Bobgy): validate executor inputs match component inputs definition
	return inputs, nil
}

// resolveInputParameter resolves an InputParameterSpec
// using a given input context via InputParams. ErrResolvedParameterNull is returned if paramSpec
// is a component input parameter and parameter resolves to a null value (i.e. an optional pipeline input with no
// default). The caller can decide if this is allowed in that context.
func resolveInputParameter(
	ctx context.Context,
	dag *metadata.DAG,
	pipeline *metadata.Pipeline,
	opts Options,
	mlmd *metadata.Client,
	paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec,
	inputParams map[string]*structpb.Value,
) (*structpb.Value, error) {
	glog.V(4).Infof("paramSpec: %v", paramSpec)
	paramError := func(err error) error {
		return fmt.Errorf("resolving input parameter with spec %s: %w", paramSpec, err)
	}
	switch t := paramSpec.Kind.(type) {
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter:
		componentInput := paramSpec.GetComponentInputParameter()
		if componentInput == "" {
			return nil, paramError(fmt.Errorf("empty component input"))
		}
		v, ok := inputParams[componentInput]
		if !ok {
			return nil, paramError(fmt.Errorf("parent DAG does not have input parameter %s", componentInput))
		}

		if _, isNullValue := v.GetKind().(*structpb.Value_NullValue); isNullValue {
			// Null values are only allowed for optional pipeline input parameters with no values. The caller has this
			// context to know if this is allowed.
			return nil, fmt.Errorf("%w: %s", ErrResolvedParameterNull, componentInput)
		}

		return v, nil

	// This is the case where the input comes from the output of an upstream task.
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter:
		cfg := resolveUpstreamOutputsConfig{
			ctx:       ctx,
			paramSpec: paramSpec,
			dag:       dag,
			pipeline:  pipeline,
			mlmd:      mlmd,
			err:       paramError,
		}
		v, err := resolveUpstreamParameters(cfg)
		if err != nil {
			return nil, err
		}
		return v, nil
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue:
		runtimeValue := paramSpec.GetRuntimeValue()
		switch t := runtimeValue.Value.(type) {
		case *pipelinespec.ValueOrRuntimeParameter_Constant:
			val := runtimeValue.GetConstant()
			var v *structpb.Value
			switch val.GetStringValue() {
			case "{{$.pipeline_job_name}}":
				v = structpb.NewStringValue(opts.RunDisplayName)
			case "{{$.pipeline_job_resource_name}}":
				v = structpb.NewStringValue(opts.RunName)
			case "{{$.pipeline_job_uuid}}":
				v = structpb.NewStringValue(opts.RunID)
			case "{{$.pipeline_task_name}}":
				v = structpb.NewStringValue(opts.Task.GetTaskInfo().GetName())
			case "{{$.pipeline_task_uuid}}":
				v = structpb.NewStringValue(fmt.Sprintf("%d", opts.DAGExecutionID))
			default:
				v = val
			}

			return v, nil
		default:
			return nil, paramError(fmt.Errorf("param runtime value spec of type %T not implemented", t))
		}
	// TODO(Bobgy): implement the following cases
	// case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_:
	default:
		return nil, paramError(fmt.Errorf("parameter spec of type %T not implemented yet", t))
	}
}

// resolveInputParameterStr is like resolveInputParameter but returns an error if the resolved value is not a non-empty
// string.
func resolveInputParameterStr(
	ctx context.Context,
	dag *metadata.DAG,
	pipeline *metadata.Pipeline,
	opts Options,
	mlmd *metadata.Client,
	paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec,
	inputParams map[string]*structpb.Value,
) (*structpb.Value, error) {
	val, err := resolveInputParameter(ctx, dag, pipeline, opts, mlmd, paramSpec, inputParams)
	if err != nil {
		return nil, err
	}

	if typedVal, ok := val.GetKind().(*structpb.Value_StringValue); ok && typedVal != nil {
		if typedVal.StringValue == "" {
			return nil, fmt.Errorf("resolving input parameter with spec %s. Expected a non-empty string.", paramSpec)
		}
	} else {
		return nil, fmt.Errorf("resolving input parameter with spec %s. Expected a string but got: %T", paramSpec, val.GetKind())
	}

	return val, nil
}

// resolveInputArtifact resolves an InputArtifactSpec
// using a given input context via inputArtifacts.
func resolveInputArtifact(
	ctx context.Context,
	dag *metadata.DAG,
	pipeline *metadata.Pipeline,
	mlmd *metadata.Client,
	name string,
	artifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec,
	inputArtifacts map[string]*pipelinespec.ArtifactList,
	task *pipelinespec.PipelineTaskSpec,
) (*pipelinespec.ArtifactList, error) {
	glog.V(4).Infof("inputs: %#v", task.GetInputs())
	glog.V(4).Infof("artifacts: %#v", task.GetInputs().GetArtifacts())
	artifactError := func(err error) error {
		return fmt.Errorf("failed to resolve input artifact %s with spec %s: %w", name, artifactSpec, err)
	}
	switch t := artifactSpec.Kind.(type) {
	case *pipelinespec.TaskInputsSpec_InputArtifactSpec_ComponentInputArtifact:
		inputArtifactName := artifactSpec.GetComponentInputArtifact()
		if inputArtifactName == "" {
			return nil, artifactError(fmt.Errorf("component input artifact key is empty"))
		}
		v, ok := inputArtifacts[inputArtifactName]
		if !ok {
			return nil, artifactError(fmt.Errorf("parent DAG does not have input artifact %s", inputArtifactName))
		}
		return v, nil
	case *pipelinespec.TaskInputsSpec_InputArtifactSpec_TaskOutputArtifact:
		cfg := resolveUpstreamOutputsConfig{
			ctx:          ctx,
			artifactSpec: artifactSpec,
			dag:          dag,
			pipeline:     pipeline,
			mlmd:         mlmd,
			err:          artifactError,
		}
		artifacts, err := resolveUpstreamArtifacts(cfg)
		if err != nil {
			return nil, err
		}
		return artifacts, nil
	default:
		return nil, artifactError(fmt.Errorf("artifact spec of type %T not implemented yet", t))
	}
}

// resolveUpstreamParameters resolves input parameters that come from upstream
// tasks. These tasks can be components/containers, which is relatively
// straightforward, or DAGs, in which case, we need to traverse the graph until
// we arrive at a component/container (since there can be n nested DAGs).
func resolveUpstreamParameters(cfg resolveUpstreamOutputsConfig) (*structpb.Value, error) {
	taskOutput := cfg.paramSpec.GetTaskOutputParameter()
	glog.V(4).Info("taskOutput: ", taskOutput)
	producerTaskName := taskOutput.GetProducerTask()
	if producerTaskName == "" {
		return nil, cfg.err(fmt.Errorf("producerTaskName is empty"))
	}
	outputParameterKey := taskOutput.GetOutputParameterKey()
	if outputParameterKey == "" {
		return nil, cfg.err(fmt.Errorf("output parameter key is empty"))
	}

	// Get a list of tasks for the current DAG first.
	// The reason we use gatDAGTasks instead of mlmd.GetExecutionsInDAG is because the latter does not handle
	// task name collisions in the map which results in a bunch of unhandled edge cases and test failures.
	tasks, err := getDAGTasks(cfg.ctx, cfg.dag, cfg.pipeline, cfg.mlmd, nil)
	if err != nil {
		return nil, cfg.err(err)
	}

	producer, ok := tasks[producerTaskName]
	if !ok {
		return nil, cfg.err(fmt.Errorf("producer task, %v, not in tasks", producerTaskName))
	}
	glog.V(4).Info("producer: ", producer)
	glog.V(4).Infof("tasks: %#v", tasks)
	currentTask := producer
	// Continue looping until we reach a sub-task that is NOT a DAG.
	for {
		glog.V(4).Info("currentTask: ", currentTask.TaskName())
		// If the current task is a DAG:
		if *currentTask.GetExecution().Type == "system.DAGExecution" {
			// Since currentTask is a DAG, we need to deserialize its
			// output parameter map so that we can look up its
			// corresponding producer sub-task, reassign currentTask,
			// and iterate through this loop again.
			outputParametersCustomProperty, ok := currentTask.GetExecution().GetCustomProperties()["parameter_producer_task"]
			if !ok {
				return nil, cfg.err(fmt.Errorf("task, %v, does not have a parameter_producer_task custom property", currentTask.TaskName()))
			}
			glog.V(4).Infof("outputParametersCustomProperty: %#v", outputParametersCustomProperty)

			dagOutputParametersMap := make(map[string]*pipelinespec.DagOutputsSpec_DagOutputParameterSpec)
			glog.V(4).Infof("outputParametersCustomProperty: %v", outputParametersCustomProperty.GetStructValue())

			for name, value := range outputParametersCustomProperty.GetStructValue().GetFields() {
				outputSpec := &pipelinespec.DagOutputsSpec_DagOutputParameterSpec{}
				err := protojson.Unmarshal([]byte(value.GetStringValue()), outputSpec)
				if err != nil {
					return nil, err
				}
				dagOutputParametersMap[name] = outputSpec
			}

			glog.V(4).Infof("Deserialized dagOutputParametersMap: %v", dagOutputParametersMap)

			// Support for the 2 DagOutputParameterSpec types:
			// ValueFromParameter & ValueFromOneof
			var subTaskName string
			switch dagOutputParametersMap[outputParameterKey].Kind.(type) {
			case *pipelinespec.DagOutputsSpec_DagOutputParameterSpec_ValueFromParameter:
				subTaskName = dagOutputParametersMap[outputParameterKey].GetValueFromParameter().GetProducerSubtask()
				outputParameterKey = dagOutputParametersMap[outputParameterKey].GetValueFromParameter().GetOutputParameterKey()
			case *pipelinespec.DagOutputsSpec_DagOutputParameterSpec_ValueFromOneof:
				// When OneOf is specified in a pipeline, the output of only 1 task is consumed even though there may be more than 1 task output set. In this case we will attempt to grab the first successful task output.
				paramSelectors := dagOutputParametersMap[outputParameterKey].GetValueFromOneof().GetParameterSelectors()
				glog.V(4).Infof("paramSelectors: %v", paramSelectors)
				// Since we have the tasks map, we can iterate through the parameterSelectors if the ProducerSubTask is not present in the task map and then assign the new OutputParameterKey only if it exists.
				successfulOneOfTask := false
				for !successfulOneOfTask {
					for _, paramSelector := range paramSelectors {
						subTaskName = paramSelector.GetProducerSubtask()
						glog.V(4).Infof("subTaskName from paramSelector: %v", subTaskName)
						glog.V(4).Infof("outputParameterKey from paramSelector: %v", paramSelector.GetOutputParameterKey())
						if subTask, ok := tasks[subTaskName]; ok {
							subTaskState := subTask.GetExecution().LastKnownState.String()
							glog.V(4).Infof("subTask: %w , subTaskState: %v", subTaskName, subTaskState)
							if subTaskState == "CACHED" || subTaskState == "COMPLETE" {

								outputParameterKey = paramSelector.GetOutputParameterKey()
								successfulOneOfTask = true
								break
							}
						}
					}
					if !successfulOneOfTask {
						return nil, cfg.err(fmt.Errorf("processing OneOf: No successful task found"))
					}
				}
			}
			glog.V(4).Infof("SubTaskName from outputParams: %v", subTaskName)
			glog.V(4).Infof("OutputParameterKey from outputParams: %v", outputParameterKey)
			if subTaskName == "" {
				return nil, cfg.err(fmt.Errorf("producer_subtask not in outputParams"))
			}
			glog.V(4).Infof(
				"Overriding currentTask, %v, output with currentTask's producer_subtask, %v, output.",
				currentTask.TaskName(),
				subTaskName,
			)
			currentTask, ok = tasks[subTaskName]
			if !ok {
				return nil, cfg.err(fmt.Errorf("subTaskName, %v, not in tasks", subTaskName))
			}
		} else {
			_, outputParametersCustomProperty, err := currentTask.GetParameters()
			if err != nil {
				return nil, err
			}
			// Base case
			return outputParametersCustomProperty[outputParameterKey], nil
		}
	}
}

// resolveUpstreamArtifacts resolves input artifacts that come from upstream
// tasks. These tasks can be components/containers, which is relatively
// straightforward, or DAGs, in which case, we need to traverse the graph until
// we arrive at a component/container (since there can be n nested DAGs).
func resolveUpstreamArtifacts(cfg resolveUpstreamOutputsConfig) (*pipelinespec.ArtifactList, error) {
	glog.V(4).Infof("artifactSpec: %#v", cfg.artifactSpec)
	taskOutput := cfg.artifactSpec.GetTaskOutputArtifact()
	if taskOutput.GetProducerTask() == "" {
		return nil, cfg.err(fmt.Errorf("producer task is empty"))
	}
	if taskOutput.GetOutputArtifactKey() == "" {
		cfg.err(fmt.Errorf("output artifact key is empty"))
	}
	tasks, err := getDAGTasks(cfg.ctx, cfg.dag, cfg.pipeline, cfg.mlmd, nil)
	if err != nil {
		cfg.err(err)
	}

	producer, ok := tasks[taskOutput.GetProducerTask()]
	if !ok {
		cfg.err(
			fmt.Errorf("cannot find producer task %q", taskOutput.GetProducerTask()),
		)
	}
	glog.V(4).Info("producer: ", producer)
	currentTask := producer
	outputArtifactKey := taskOutput.GetOutputArtifactKey()

	// Continue looping until we reach a sub-task that is NOT a DAG.
	for {
		glog.V(4).Info("currentTask: ", currentTask.TaskName())
		// If the current task is a DAG:
		if *currentTask.GetExecution().Type == "system.DAGExecution" {
			// Get the sub-task.
			outputArtifactsCustomProperty := currentTask.GetExecution().GetCustomProperties()["artifact_producer_task"]
			// Deserialize the output artifacts.
			var outputArtifacts map[string]*pipelinespec.DagOutputsSpec_DagOutputArtifactSpec
			err := json.Unmarshal([]byte(outputArtifactsCustomProperty.GetStringValue()), &outputArtifacts)
			if err != nil {
				return nil, err
			}
			glog.V(4).Infof("Deserialized outputArtifacts: %v", outputArtifacts)
			// Adding support for multiple output artifacts
			var subTaskName string
			artifactSelectors := outputArtifacts[outputArtifactKey].GetArtifactSelectors()

			for _, v := range artifactSelectors {
				glog.V(4).Infof("v: %v", v)
				glog.V(4).Infof("v.ProducerSubtask: %v", v.ProducerSubtask)
				glog.V(4).Infof("v.OutputArtifactKey: %v", v.OutputArtifactKey)
				subTaskName = v.ProducerSubtask
				outputArtifactKey = v.OutputArtifactKey
			}
			// If the sub-task is a DAG, reassign currentTask and run
			// through the loop again.
			currentTask = tasks[subTaskName]
			// }
		} else {
			// Base case, currentTask is a container, not a DAG.
			outputs, err := cfg.mlmd.GetOutputArtifactsByExecutionId(cfg.ctx, currentTask.GetID())
			if err != nil {
				cfg.err(err)
			}
			glog.V(4).Infof("outputs: %#v", outputs)
			artifact, ok := outputs[outputArtifactKey]
			if !ok {
				cfg.err(
					fmt.Errorf(
						"cannot find output artifact key %q in producer task %q",
						taskOutput.GetOutputArtifactKey(),
						taskOutput.GetProducerTask(),
					),
				)
			}
			runtimeArtifact, err := artifact.ToRuntimeArtifact()
			if err != nil {
				cfg.err(err)
			}
			// Base case
			return &pipelinespec.ArtifactList{
				Artifacts: []*pipelinespec.RuntimeArtifact{runtimeArtifact},
			}, nil
		}
	}
}

// getDAGTasks is a recursive function that returns a map of all tasks across all DAGs in the context of nested DAGs.
func getDAGTasks(
	ctx context.Context,
	dag *metadata.DAG,
	pipeline *metadata.Pipeline,
	mlmd *metadata.Client,
	flattenedTasks map[string]*metadata.Execution,
) (map[string]*metadata.Execution, error) {
	if flattenedTasks == nil {
		flattenedTasks = make(map[string]*metadata.Execution)
	}
	currentExecutionTasks, err := mlmd.GetExecutionsInDAG(ctx, dag, pipeline, true)
	if err != nil {
		return nil, err
	}
	for k, v := range currentExecutionTasks {
		flattenedTasks[k] = v
	}
	for _, v := range currentExecutionTasks {

		if v.GetExecution().GetType() == "system.DAGExecution" {
			// Iteration count is only applied when using ParallelFor, and in
			// that scenario you're guaranteed to have redundant task names even
			// within a single DAG, which results in an error when
			// mlmd.GetExecutionsInDAG is called. ParallelFor outputs should be
			// handled with dsl.Collected.
			_, ok := v.GetExecution().GetCustomProperties()["iteration_count"]
			if ok {
				glog.Infof("Found a ParallelFor task, %v. Skipping it.", v.TaskName())
				continue
			}
			glog.V(4).Infof("Found a task, %v, with an execution type of system.DAGExecution. Adding its tasks to the task list.", v.TaskName())
			subDAG, err := mlmd.GetDAG(ctx, v.GetExecution().GetId())
			if err != nil {
				return nil, err
			}
			// Pass the subDAG into a recursive call to getDAGTasks and update
			// tasks to include the subDAG's tasks.
			flattenedTasks, err = getDAGTasks(ctx, subDAG, pipeline, mlmd, flattenedTasks)
			if err != nil {
				return nil, err
			}
		}
	}

	return flattenedTasks, nil
}
