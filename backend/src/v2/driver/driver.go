// Copyright 2021-2023 The Kubeflow Authors
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
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_v2"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"google.golang.org/protobuf/types/known/structpb"
)

var dummyImages = map[string]string{
	"argostub/createpvc": "create PVC",
	"argostub/deletepvc": "delete PVC",
}

// Driver options
type Options struct {
	// required, pipeline context name
	PipelineName string
	// required, KFP run ID
	RunID string
	// required, Component spec
	Component *pipelinespec.ComponentSpec
	// optional, iteration index. -1 means not an iteration.
	IterationIndex int

	// optional, required only by root DAG driver
	RuntimeConfig *pipelinespec.PipelineJob_RuntimeConfig
	Namespace     string

	// optional, required by non-root drivers
	Task           *pipelinespec.PipelineTaskSpec
	DAGExecutionID int64

	// optional, required only by container driver
	Container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec

	// optional, allows to specify kubernetes-specific executor config
	KubernetesExecutorConfig *kubernetesplatform.KubernetesExecutorConfig

	// optional, required only if the {{$.pipeline_job_resource_name}} placeholder is used
	RunName string
	// optional, required only if the {{$.pipeline_job_name}} placeholder is used
	RunDisplayName string

	PipelineLogLevel string

	PublishLogs    string
	MetadataClient metadata_v2.MetadataInterfaceClient

	ExperimentId string

	DevMode        bool
	DevExecutionId int64
}

// Identifying information used for error messages
func (o Options) info() string {
	msg := fmt.Sprintf("pipelineName=%v, runID=%v", o.PipelineName, o.RunID)
	if o.Task.GetTaskInfo().GetName() != "" {
		msg = msg + fmt.Sprintf(", task=%q", o.Task.GetTaskInfo().GetName())
	}
	if o.Task.GetComponentRef().GetName() != "" {
		msg = msg + fmt.Sprintf(", component=%q", o.Task.GetComponentRef().GetName())
	}
	if o.DAGExecutionID != 0 {
		msg = msg + fmt.Sprintf(", dagExecutionID=%v", o.DAGExecutionID)
	}
	if o.IterationIndex >= 0 {
		msg = msg + fmt.Sprintf(", iterationIndex=%v", o.IterationIndex)
	}
	if o.RuntimeConfig != nil {
		msg = msg + ", runtimeConfig" // this only means runtimeConfig is not empty
	}
	if o.Component.GetImplementation() != nil {
		msg = msg + ", componentSpec" // this only means componentSpec is not empty
	}
	if o.KubernetesExecutorConfig != nil {
		msg = msg + ", KubernetesExecutorConfig" // this only means KubernetesExecutorConfig is not empty
	}
	return msg
}

type Execution struct {
	ID             int64
	ExecutorInput  *pipelinespec.ExecutorInput
	IterationCount *int  // number of iterations, -1 means not an iterator
	Condition      *bool // true -> trigger the task, false -> not trigger the task, nil -> the task is unconditional

	// only specified when this is a Container execution
	Cached       *bool
	PodSpecPatch string
}

func (e *Execution) WillTrigger() bool {
	if e == nil || e.Condition == nil {
		return true
	}
	return *e.Condition
}

// Get iteration items from a structpb.Value.
// Return value may be
// * a list of JSON serializable structs
// * a list of structpb.Value
func getItems(value *structpb.Value) (items []*structpb.Value, err error) {
	switch v := value.GetKind().(type) {
	case *structpb.Value_ListValue:
		return v.ListValue.GetValues(), nil
	case *structpb.Value_StringValue:
		listValue := structpb.Value{}
		if err = listValue.UnmarshalJSON([]byte(v.StringValue)); err != nil {
			return nil, err
		}
		return listValue.GetListValue().GetValues(), nil
	default:
		return nil, fmt.Errorf("value of type %T cannot be iterated", v)
	}
}
