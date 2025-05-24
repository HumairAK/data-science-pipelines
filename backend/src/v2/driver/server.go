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
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/plugins/executor"
	"net/http"

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
		glog.Errorf("Failed to marshal request to JSON: %v", err)
	} else {
		glog.Infof("Received request: %s", string(prettyJSON))
	}

	// TODO: It would be nice if instead of "parameters"
	// we could have a "driver" field in the plugin object.
	// and then we can just unmarshal the driver field into
	// the Options struct.
	options := Options{}
	parameters := argoTemplateArgs.Template.Inputs.Parameters
	for _, parameter := range parameters {
		if parameter.Name == "pipeline_name" {
			options.PipelineName = parameter.Value.String()
		}
	}

	//ctx := context.Background()
	//var execution *Execution
	//switch req.DriverType {
	//case "ROOT_DAG":
	//	execution, err = RootDAG(ctx, options, s.mlmdClient)
	//case "DAG":
	//	execution, err = DAG(ctx, options, s.mlmdClient)
	//case "CONTAINER":
	//	execution, err = Container(ctx, options, s.mlmdClient, s.cacheClient)
	//default:
	//	http.Error(w, fmt.Sprintf("Unknown driver type: %s", req.DriverType), http.StatusBadRequest)
	//	return
	//}

	if err != nil {
		http.Error(w, fmt.Sprintf("Driver execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := executor.ExecuteTemplateResponse{
		Body: executor.ExecuteTemplateReply{
			Node: &v1alpha1.NodeResult{
				Phase:   "succeeded",
				Message: "hello tempalte!",
				Outputs: &v1alpha1.Outputs{
					Parameters: []v1alpha1.Parameter{
						{
							Name:  "pod-spec-patch",
							Value: v1alpha1.AnyStringPtr("some json"),
						},
					},
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
