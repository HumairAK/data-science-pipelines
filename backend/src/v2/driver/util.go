// Copyright 2021-2024 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"regexp"
)

// inputPipelineChannelPattern define a regex pattern to match the content within single quotes
// example input channel looks like "{{$.inputs.parameters['pipelinechannel--val']}}"
const inputPipelineChannelPattern = `\$.inputs.parameters\['(.+?)'\]`

func isInputParameterChannel(inputChannel string) bool {
	re := regexp.MustCompile(inputPipelineChannelPattern)
	match := re.FindStringSubmatch(inputChannel)
	if len(match) == 2 {
		return true
	} else {
		// if len(match) > 2, then this is still incorrect because
		// inputChannel should contain only one parameter channel input
		return false
	}
}

// extractInputParameterFromChannel takes an inputChannel that adheres to
// inputPipelineChannelPattern and extracts the channel parameter name.
// For example given an input channel of the form "{{$.inputs.parameters['pipelinechannel--val']}}"
// the channel parameter name "pipelinechannel--val" is returned.
func extractInputParameterFromChannel(inputChannel string) (string, error) {
	re := regexp.MustCompile(inputPipelineChannelPattern)
	match := re.FindStringSubmatch(inputChannel)
	if len(match) > 1 {
		extractedValue := match[1]
		return extractedValue, nil
	} else {
		return "", fmt.Errorf("failed to extract input parameter from channel: %s", inputChannel)
	}
}

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

func resolveK8sParameter(
	ctx context.Context,
	dag *metadata.DAG,
	pipeline *metadata.Pipeline,
	mlmd *metadata.Client,
	k8sParamSpec *kubernetesplatform.InputParameterSpec,
	inputParams map[string]*structpb.Value,
) (*structpb.Value, error) {
	pipelineParamSpec := &pipelinespec.TaskInputsSpec_InputParameterSpec{}
	err := convertToProtoMessages(k8sParamSpec, pipelineParamSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input parameter spec to pipeline spec: %v", err)
	}
	resolvedSecretName, err := resolveParameter(ctx, dag, pipeline, mlmd, pipelineParamSpec, inputParams)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve input parameter name: %w", err)
	}
	return resolvedSecretName, nil
}

func resolveParameter(
	ctx context.Context,
	dag *metadata.DAG,
	pipeline *metadata.Pipeline,
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
			return runtimeValue.GetConstant(), nil
		default:
			return nil, paramError(fmt.Errorf("param runtime value spec of type %T not implemented", t))
		}
	// TODO(Bobgy): implement the following cases
	// case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_:
	default:
		return nil, paramError(fmt.Errorf("parameter spec of type %T not implemented yet", t))
	}
}

func convertToProtoMessages(src *kubernetesplatform.InputParameterSpec, dst *pipelinespec.TaskInputsSpec_InputParameterSpec) error {
	data, err := protojson.Marshal(proto.MessageV2(src))
	if err != nil {
		return err
	}
	return protojson.Unmarshal(data, proto.MessageV2(dst))
}
