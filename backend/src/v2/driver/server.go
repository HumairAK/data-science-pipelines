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
	"net/http"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
)

// TemplateExecuteRequest represents the request body for the template.execute endpoint
type TemplateExecuteRequest struct {
	PipelineName     string                                 `json:"pipeline_name"`
	RunID            string                                 `json:"run_id"`
	RunName          string                                 `json:"run_name"`
	RunDisplayName   string                                 `json:"run_display_name"`
	Namespace        string                                 `json:"namespace"`
	Component        *pipelinespec.ComponentSpec            `json:"component"`
	Task             *pipelinespec.PipelineTaskSpec         `json:"task,omitempty"`
	DAGExecutionID   int64                                  `json:"dag_execution_id,omitempty"`
	IterationIndex   int                                    `json:"iteration_index,omitempty"`
	Container        *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec `json:"container,omitempty"`
	RuntimeConfig    *pipelinespec.PipelineJob_RuntimeConfig `json:"runtime_config,omitempty"`
	DriverType       string                                 `json:"driver_type"`
	PipelineLogLevel string                                 `json:"pipeline_log_level,omitempty"`
	PublishLogs      string                                 `json:"publish_logs,omitempty"`
	CacheDisabled    bool                                   `json:"cache_disabled,omitempty"`
}

// TemplateExecuteResponse represents the response body for the template.execute endpoint
type TemplateExecuteResponse struct {
	ExecutionID    int64                      `json:"execution_id,omitempty"`
	IterationCount *int                       `json:"iteration_count,omitempty"`
	Cached         *bool                      `json:"cached,omitempty"`
	Condition      *bool                      `json:"condition,omitempty"`
	PodSpecPatch   string                     `json:"pod_spec_patch,omitempty"`
	ExecutorInput  *pipelinespec.ExecutorInput `json:"executor_input,omitempty"`
}

// Server represents the REST server for the driver component
type Server struct {
	mlmdClient   *metadata.Client
	cacheClient  cacheutils.Client
	port         int
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

	var req TemplateExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	options := Options{
		PipelineName:     req.PipelineName,
		RunID:            req.RunID,
		RunName:          req.RunName,
		RunDisplayName:   req.RunDisplayName,
		Namespace:        req.Namespace,
		Component:        req.Component,
		Task:             req.Task,
		DAGExecutionID:   req.DAGExecutionID,
		IterationIndex:   req.IterationIndex,
		Container:        req.Container,
		RuntimeConfig:    req.RuntimeConfig,
		PipelineLogLevel: req.PipelineLogLevel,
		PublishLogs:      req.PublishLogs,
		CacheDisabled:    req.CacheDisabled,
	}

	var execution *Execution
	var err error

	switch req.DriverType {
	case "ROOT_DAG":
		execution, err = RootDAG(ctx, options, s.mlmdClient)
	case "DAG":
		execution, err = DAG(ctx, options, s.mlmdClient)
	case "CONTAINER":
		execution, err = Container(ctx, options, s.mlmdClient, s.cacheClient)
	default:
		http.Error(w, fmt.Sprintf("Unknown driver type: %s", req.DriverType), http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Driver execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := TemplateExecuteResponse{
		ExecutionID:    execution.ID,
		IterationCount: execution.IterationCount,
		Cached:         execution.Cached,
		Condition:      execution.Condition,
		PodSpecPatch:   execution.PodSpecPatch,
		ExecutorInput:  execution.ExecutorInput,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}
