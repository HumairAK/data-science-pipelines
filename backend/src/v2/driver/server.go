// Copyright 2025 The Kubeflow Authors
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
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/plugins/executor"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/v2Util"
	"net/http"
	"strconv"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
)

const driverPluginName = "driver-argo-executor"

// Server represents the REST server for the driver component
type Server struct {
	mlmdClient  *metadata.Client
	cacheClient cacheutils.Client
	port        int
}

// NewServer creates a new Server instance
func NewServer(mlmdClient *metadata.Client, cacheClient cacheutils.Client, port int) *Server {
	return &Server{
		mlmdClient:  mlmdClient,
		cacheClient: cacheClient,
		port:        port,
	}
}

// Start starts the REST server
func (s *Server) Start() error {
	http.HandleFunc("/api/v1/template.execute", s.handleTemplateExecute)

	glog.Infof("Starting driver REST server on port %d", s.port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)
}

// handleTemplateExecute handles the /api/v1/template.execute endpoint
func (s *Server) handleTemplateExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var argoTemplateArgs executor.ExecuteTemplateArgs
	if err := json.NewDecoder(r.Body).Decode(&argoTemplateArgs); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	pluginObject := argoTemplateArgs.Template.Plugin
	// First, make sure plugin is not nil
	if pluginObject == nil {
		http.Error(w, fmt.Sprintf("plugin was empty"), http.StatusBadRequest)
		return
	}
	m := map[string]interface{}{}
	if err := json.Unmarshal(pluginObject.Object.Value, &m); err != nil {
		http.Error(w, fmt.Sprintf("failed to unmarshal plugin object value to map[string]interface{}"), http.StatusBadRequest)
		return
	}
	// Check if "driver" key exists in the map
	_, exists := m[driverPluginName]
	if !exists {
		http.Error(w, "plugin object value is missing required 'driver' field", http.StatusBadRequest)
		return
	}

	glog.Infof("Received plugin request for plugin %s", driverPluginName)
	// Print the decoded JSON request
	prettyJSON, err := json.MarshalIndent(argoTemplateArgs, "", "    ")
	if err != nil {
		http.Error(w, "Failed to marshal request to JSON", http.StatusBadRequest)
		return
	} else {
		glog.Infof("Received request: %s", string(prettyJSON))
	}

	// TODO: It would be nice if instead of "parameters"
	// we could have a "driver" field in the plugin object.
	// and then we can just unmarshal the driver field into
	// the Options struct.
	// however parameters like component come from the workflow and
	// are not est at the compiler time
	// we will need to split those up
	options := Options{}
	parameters := argoTemplateArgs.Template.Inputs.Parameters
	for _, parameter := range parameters {
		switch parameter.Name {
		case "component":
			componentJson := parameter.Value.String()
			glog.Infof("input componentJson:%s\n", v2Util.PrettyPrint(componentJson))
			componentSpec := &pipelinespec.ComponentSpec{}
			err1 := util.UnmarshalString(componentJson, componentSpec)
			if err1 != nil {
				http.Error(w, fmt.Sprintf("failed to unmarshal component json: %v", err1), http.StatusBadRequest)
				return
			}
			options.Component = componentSpec
		case "runtime-config":
			runtimeConfigJson := parameter.Value.String()
			if runtimeConfigJson != "" {
				glog.Infof("input RuntimeConfig:%s\n", v2Util.PrettyPrint(runtimeConfigJson))
				runtimeConfig := &pipelinespec.PipelineJob_RuntimeConfig{}
				err1 := util.UnmarshalString(runtimeConfigJson, runtimeConfig)
				if err1 != nil {
					http.Error(w, fmt.Sprintf("failed to unmarshal component json: %v", err1), http.StatusBadRequest)
					return
				}
				options.RuntimeConfig = runtimeConfig
			}
		case "task":
			taskSpecJson := parameter.Value.String()
			if taskSpecJson != "" {
				glog.Infof("input TaskSpec:%s\n", v2Util.PrettyPrint(taskSpecJson))
				taskSpec := &pipelinespec.PipelineTaskSpec{}
				err1 := util.UnmarshalString(taskSpecJson, taskSpec)
				if err1 != nil {
					http.Error(w, fmt.Sprintf("failed to unmarshal component json: %v", err1), http.StatusBadRequest)
					return
				}
				options.Task = taskSpec
			}
		case "parent-dag-id":
			dagID, err1 := strconv.ParseInt(parameter.Value.String(), 10, 64)
			if err1 != nil {
				http.Error(w, fmt.Sprintf("failed to parse parent-dag-id: %v", err1), http.StatusBadRequest)
				return
			}
			options.DAGExecutionID = dagID
		case "iteration-index":
			iterIndex, err1 := strconv.ParseInt(parameter.Value.String(), 10, 64)
			if err1 != nil {
				http.Error(w, fmt.Sprintf("failed to parse iteration-index: %v", err1), http.StatusBadRequest)
				return
			}
			options.IterationIndex = int(iterIndex)
		case "driver-type":
			options.DriverType = parameter.Value.String()
		case "pipeline_name":
			options.PipelineName = parameter.Value.String()
		case "run_id":
			options.RunID = parameter.Value.String()
		case "run_name":
			options.RunName = parameter.Value.String()
		case "run_display_name":
			options.RunDisplayName = parameter.Value.String()
		case "http_proxy":
			options.RunName = parameter.Value.String()
		case "https_proxy":
			options.RunName = parameter.Value.String()
		case "no_proxy":
			options.RunName = parameter.Value.String()
		case "cache_disabled":
			cacheDisabled, err1 := strconv.ParseBool(parameter.Value.String())
			if err1 != nil {
				http.Error(w, fmt.Sprintf("failed to parse cache_disabled: %v", err1), http.StatusBadRequest)
				return
			}
			options.CacheDisabled = cacheDisabled
		case "log_level":
			options.PipelineLogLevel = parameter.Value.String()
		case "publish_logs":
			options.PublishLogs = parameter.Value.String()
		// Container dag:
		case "container":
			containerSpecJson := parameter.Value.String()
			glog.Infof("input ContainerSpec:%s\n", v2Util.PrettyPrint(containerSpecJson))
			containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{}
			err1 := util.UnmarshalString(containerSpecJson, containerSpec)
			if err1 != nil {
				http.Error(w, fmt.Sprintf("failed to unmarshal component json: %v", err1), http.StatusBadRequest)
				return
			}
			options.Container = containerSpec
		case "kubernetes-config":
			k8sExecConfigJson := parameter.Value.String()
			k8sExecCfg, err1 := v2Util.ParseExecConfigJson(&k8sExecConfigJson)
			if err1 != nil {
				http.Error(w, fmt.Sprintf("encountered issues when parsing kube config: %v", err1), http.StatusBadRequest)
				return
			}
			options.KubernetesExecutorConfig = k8sExecCfg
		}
	}

	ctx := context.Background()
	var execution *Execution
	switch options.DriverType {
	case "ROOT_DAG":
		execution, err = RootDAG(ctx, options, s.mlmdClient)
	case "DAG":
		execution, err = DAG(ctx, options, s.mlmdClient)
	case "CONTAINER":
		execution, err = Container(ctx, options, s.mlmdClient, s.cacheClient)
	default:
		http.Error(w, fmt.Sprintf("Unknown driver type: %s", options.DriverType), http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Driver execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	outputParameters, err := appendExecutionParameters(execution, options)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to append execution parameters: %v", err), http.StatusInternalServerError)
		return
	}

	response := executor.ExecuteTemplateResponse{
		Body: executor.ExecuteTemplateReply{
			Node: &v1alpha1.NodeResult{
				Phase: "succeeded",
				Message: fmt.Sprintf("Driver call for driver type %s completed successfully. Execution ID: %d",
					options.DriverType, execution.ID),
				Outputs: &v1alpha1.Outputs{
					Parameters: outputParameters,
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}
