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
	"fmt"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"regexp"
)

// InputPipelineChannelPattern define a regex pattern to match the content within single quotes
// example input channel loooks like "{{$.inputs.parameters[pipelinechannel--img]}}"
const InputPipelineChannelPattern = `\$.inputs.parameters\['(.+?)'\]`

func isInputParameterChannel(inputChannel string) bool {
	re := regexp.MustCompile(InputPipelineChannelPattern)
	match := re.FindStringSubmatch(inputChannel)
	if len(match) == 2 {
		return true
	} else {
		// if len(match) > 1, then this is still incorrect because
		// inputChannel should contain only one parameter channel input
		return false
	}
}

func extractInputParameterFromChannel(inputChannel string) (string, error) {
	// inside `$.inputs.parameters[]`
	// Compile the regex
	re := regexp.MustCompile(InputPipelineChannelPattern)
	// Find the first match
	match := re.FindStringSubmatch(inputChannel)
	if len(match) > 1 {
		// match[1] contains the value inside the single quotes
		extractedValue := match[1]
		return extractedValue, nil
	} else {
		return "", fmt.Errorf("failed to extract input parameter from channel: %s", inputChannel)
	}
}

// resolvePodSpecRuntimeParameter resolves runtime value that is intended to be
// utilized within the Pod Spec. parameterValue takes the form of:
// "{{$.inputs.parameters[pipelinechannel--someParameterName]}}"
// these are runtime parameter values that have been resolved and included within
// the executor input. Since the pod spec patch cannot dynamically update the underlying
// container template's inputs in an Argo Workflow, this is a workaround for resolving
// such parameters.
//
// If parameter value is not a parameter channel, then a constant value is assumed and
// returned as is.
func resolvePodSpecRuntimeParameter(parameterValue string, executorInput *pipelinespec.ExecutorInput) (string, error) {
	if isInputParameterChannel(parameterValue) {
		inputImage, err1 := extractInputParameterFromChannel(parameterValue)
		if err1 != nil {
			return "", err1
		}
		glog.Infof("Got inputImage %s", inputImage)
		if val, ok := executorInput.Inputs.ParameterValues[inputImage]; ok {
			return val.GetStringValue(), nil
		} else {
			glog.Infof("ParameterValues %s", executorInput.Inputs.ParameterValues)
			return "", fmt.Errorf("executorInput did not contain container Image input parameter")
		}
	}
	return parameterValue, nil
}
