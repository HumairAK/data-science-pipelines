package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"k8s.io/client-go/kubernetes"
)

func validateRootDAG(opts Options) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("invalid root DAG driver args: %w", err)
		}
	}()
	if opts.PipelineName == "" {
		return fmt.Errorf("pipeline name is required")
	}
	if opts.RunID == "" {
		return fmt.Errorf("KFP run ID is required")
	}
	if opts.Component == nil {
		return fmt.Errorf("component spec is required")
	}
	if opts.RuntimeConfig == nil {
		return fmt.Errorf("runtime config is required")
	}
	if opts.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if opts.Task.GetTaskInfo().GetName() != "" {
		return fmt.Errorf("task spec is unnecessary")
	}
	if opts.DAGExecutionID != 0 {
		return fmt.Errorf("DAG execution ID is unnecessary")
	}
	if opts.Container != nil {
		return fmt.Errorf("container spec is unnecessary")
	}
	if opts.IterationIndex >= 0 {
		return fmt.Errorf("iteration index is unnecessary")
	}
	return nil
}

func RootDAG(ctx context.Context, opts Options, mlmd *metadata.Client) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.RootDAG(%s) failed: %w", opts.info(), err)
		}
	}()
	b, _ := json.Marshal(opts)
	glog.V(4).Info("RootDAG opts: ", string(b))
	err = validateRootDAG(opts)
	if err != nil {
		return nil, err
	}
	pipelineRoot := opts.RuntimeConfig.GetGcsOutputDirectory()

	restConfig, err := util.GetKubernetesConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client: %w", err)
	}
	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client set: %w", err)
	}
	cfg, err := config.FromConfigMap(ctx, k8sClient, opts.Namespace)
	if err != nil {
		return nil, err
	}

	storeSessionInfo := objectstore.SessionInfo{}
	if pipelineRoot != "" {
		glog.Infof("PipelineRoot=%q", pipelineRoot)
	} else {
		pipelineRoot = cfg.DefaultPipelineRoot()
		glog.Infof("PipelineRoot=%q from default config", pipelineRoot)
	}
	storeSessionInfo, err = cfg.GetStoreSessionInfo(pipelineRoot)
	if err != nil {
		return nil, err
	}
	storeSessionInfoJSON, err := json.Marshal(storeSessionInfo)
	if err != nil {
		return nil, err
	}
	storeSessionInfoStr := string(storeSessionInfoJSON)

	pipeline, err := mlmd.GetPipeline(ctx, opts.PipelineName, opts.RunID, opts.Namespace, "run-resource", pipelineRoot, storeSessionInfoStr)
	if err != nil {
		return nil, err
	}

	_, err = opts.MetadataClient.CreatePipelineRun(ctx, opts.PipelineName, opts.RunID, opts.Namespace, "run-resource", pipelineRoot, storeSessionInfoStr)
	if err != nil {
		return nil, err
	}

	executorInput := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			ParameterValues: opts.RuntimeConfig.GetParameterValues(),
		},
	}
	// TODO(Bobgy): validate executorInput matches component spec types
	ecfg, err := metadata.GenerateExecutionConfig(executorInput)
	if err != nil {
		return nil, err
	}
	ecfg.ExecutionType = metadata.DagExecutionTypeName
	ecfg.Name = fmt.Sprintf("run/%s", opts.RunID)
	exec, err := mlmd.CreateExecution(ctx, pipeline, ecfg)
	if err != nil {
		return nil, err
	}
	glog.Infof("Created execution: %s", exec)
	// No need to return ExecutorInput, because tasks in the DAG will resolve
	// needed info from MLMD.
	return &Execution{ID: exec.GetID()}, nil
}
