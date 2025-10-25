// Copyright 2021-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"gocloud.dev/blob"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type LauncherV2Options struct {
	Namespace         string
	PodName           string
	PodUID            string
	PipelineName      string
	PublishLogs       string
	CachedFingerprint string
	CacheDisabled     bool
	IterationIndex    *int64
	ComponentSpec     *pipelinespec.ComponentSpec
	ImporterSpec      *pipelinespec.PipelineDeploymentConfig_ImporterSpec
	TaskSpec          *pipelinespec.PipelineTaskSpec
	ScopePath         util.ScopePath
	Run               *apiV2beta1.Run
	ParentTask        *apiV2beta1.PipelineTaskDetail
	Task              *apiV2beta1.PipelineTaskDetail
}

type LauncherV2 struct {
	executorInput *pipelinespec.ExecutorInput
	command       string
	args          []string
	options       LauncherV2Options
	clientManager client_manager.ClientManagerInterface
	// Maintaining a cache of opened buckets will minimize
	// the number of calls to the object store, and api server
	openedBucketCache map[string]*blob.Bucket
	launcherConfig    *config.Config

	// Dependency interfaces for testing
	fileSystem  FileSystem
	cmdExecutor CommandExecutor
	objectStore ObjectStoreClient
}

// NewLauncherV2 is a factory function that returns an instance of LauncherV2.
func NewLauncherV2(
	executorInputJSON string,
	cmdArgs []string,
	opts *LauncherV2Options,
	clientManager client_manager.ClientManagerInterface,
) (l *LauncherV2, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to create component launcher v2: %w", err)
		}
	}()

	executorInput := &pipelinespec.ExecutorInput{}
	err = protojson.Unmarshal([]byte(executorInputJSON), executorInput)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal executor input: %w", err)
	}
	if len(cmdArgs) == 0 {
		return nil, fmt.Errorf("command and arguments are empty")
	}
	err = opts.validate()
	if err != nil {
		return nil, err
	}
	launcher := &LauncherV2{
		executorInput: executorInput,
		command:       cmdArgs[0],
		args:          cmdArgs[1:],
		options:       *opts,
		clientManager: clientManager,
		// Initialize with production implementations
		fileSystem:  &OSFileSystem{},
		cmdExecutor: &RealCommandExecutor{},
	}
	// Object store is initialized after launcher creation
	launcher.objectStore = NewRealObjectStoreClient(launcher)
	return launcher, nil
}

// WithFileSystem allows overriding the file system (for testing)
func (l *LauncherV2) WithFileSystem(fs FileSystem) *LauncherV2 {
	l.fileSystem = fs
	return l
}

// WithCommandExecutor allows overriding the command executor (for testing)
func (l *LauncherV2) WithCommandExecutor(executor CommandExecutor) *LauncherV2 {
	l.cmdExecutor = executor
	return l
}

// WithObjectStore allows overriding the object store client (for testing)
func (l *LauncherV2) WithObjectStore(store ObjectStoreClient) *LauncherV2 {
	l.objectStore = store
	return l
}

// stopWaitingArtifacts will create empty files to tell Modelcar sidecar containers to stop. Any errors encountered are
// logged since this is meant as a deferred function at the end of the launcher's execution.
func stopWaitingArtifacts(artifacts map[string]*pipelinespec.ArtifactList) {
	for _, artifactList := range artifacts {
		if len(artifactList.Artifacts) == 0 {
			continue
		}

		// Following the convention of downloadArtifacts in the launcher to only look at the first in the list.
		for _, artifact := range artifactList.Artifacts {
			inputArtifact := artifact

			// This should ideally verify that this is also a model input artifact, but this metadata doesn't seem to
			// be set on inputArtifact.
			if !strings.HasPrefix(inputArtifact.Uri, "oci://") {
				continue
			}

			localPath, err := LocalPathForURI(inputArtifact.Uri)
			if err != nil {
				continue
			}

			glog.Infof("Stopping Modelcar container for artifact %s", inputArtifact.Uri)

			launcherCompleteFile := strings.TrimSuffix(localPath, "/models") + "/launcher-complete"
			_, err = os.Create(launcherCompleteFile)
			if err != nil {
				glog.Errorf(
					"Failed to stop the artifact %s by creating %s: %v", inputArtifact.Uri, launcherCompleteFile, err,
				)

				continue
			}
		}
	}
}

// updateStatuses Traverse up the dag until we find a parent task that still has other children with "RUNNING" status
// or when we have reached Root. If the parent task has other children in running that means this parent is also running.
// However, if the currentTask is a parent task, and all children tasks have been created (though not necessarily completed):
//   - if all children in this DAG are all CACHED, then the currentTask should be updated to be "CACHED"
//   - if any of the children in this DAG are FAILED, then the currentTask should be updated to be "FAILED"
//   - if all children in this DAG were SKIPPED, then the currentTask should be updated to be "SKIPPED"
//   - In any other case the state is SUCCEEDED
func updateStatuses(ctx context.Context, kfpAPIClient kfpapi.API, run *apiV2beta1.Run, currentTask *apiV2beta1.PipelineTaskDetail) error {
	// Create a map of task IDs to tasks for quick lookup
	taskMap := make(map[string]*apiV2beta1.PipelineTaskDetail)
	for _, task := range run.GetTasks() {
		taskMap[task.GetTaskId()] = task
	}

	// Start with the current task and traverse up
	for {
		// If current task has no parent, we've reached the root
		if currentTask.ParentTaskId == nil || *currentTask.ParentTaskId == "" {
			// Evaluate the root task's status based on its children
			if err := evaluateAndUpdateParentStatus(ctx, kfpAPIClient, run, currentTask); err != nil {
				return err
			}
			break
		}

		// Get the parent task
		parentTask, exists := taskMap[*currentTask.ParentTaskId]
		if !exists {
			return fmt.Errorf("parent task %s not found for task %s", *currentTask.ParentTaskId, currentTask.GetTaskId())
		}

		// Before we proceed to updating this parent task's status, we need to ensure that all child tasks have been
		// created (irrespective of their status).
		var expectedTotalChildTasks int
		if parentTask.GetType() == apiV2beta1.PipelineTaskDetail_LOOP {
			typeAttrs := parentTask.GetTypeAttributes()
			if typeAttrs == nil || typeAttrs.IterationCount == nil {
				return fmt.Errorf("loop task %s is missing iteration_count attribute", parentTask.GetTaskId())
			}
			expectedTotalChildTasks = int(*typeAttrs.IterationCount)
		} else {
			// In a non-loop case we can determine the total number of child tasks by inspecting the parent dag's
			// task count within it's component spec.
			// We need to use the parent task's scope path, not the current task's scope path
			getScopePath, err := util.ScopePathFromStringPath(run.GetPipelineSpec(), parentTask.GetScopePath())
			if err != nil {
				return fmt.Errorf("failed to get scope path for parent task %s: %w", parentTask.GetTaskId(), err)
			}
			if getScopePath.GetLast() == nil || getScopePath.GetLast().GetComponentSpec() == nil || getScopePath.GetLast().GetComponentSpec().GetDag() == nil {
				return fmt.Errorf("failed to get dag for parent task %s (scope: %s): component spec or dag is nil", parentTask.GetTaskId(), parentTask.GetScopePath())
			}
			getScopePath.GetLast().GetComponentSpec().GetDag().GetTasks()
			expectedTotalChildTasks = len(getScopePath.GetLast().GetComponentSpec().GetDag().GetTasks())
		}
		// Now count the actual number of child tasks created.
		var childCount int
		for _, task := range run.GetTasks() {
			if task.ParentTaskId != nil && *task.ParentTaskId == parentTask.GetTaskId() {
				if task.GetStatus() == apiV2beta1.PipelineTaskDetail_RUNNING {
					return nil
				}
				childCount++
			}
		}

		// If not all children created yet, exit traversal
		if childCount != expectedTotalChildTasks {
			return nil
		}

		// Evaluate and update parent's status based on its children
		if err := evaluateAndUpdateParentStatus(ctx, kfpAPIClient, run, parentTask); err != nil {
			return err
		}

		// Move to the parent for next iteration
		currentTask = parentTask
	}

	return nil
}

// evaluateAndUpdateParentStatus evaluates a parent task's status based on its direct children and updates it accordingly
func evaluateAndUpdateParentStatus(
	ctx context.Context,
	kfpAPIClient kfpapi.API,
	run *apiV2beta1.Run,
	parentTask *apiV2beta1.PipelineTaskDetail,
) error {
	// Collect all direct children of this parent
	var children []*apiV2beta1.PipelineTaskDetail
	for _, task := range run.GetTasks() {
		if task.ParentTaskId != nil && *task.ParentTaskId == parentTask.GetTaskId() {
			children = append(children, task)
		}
	}

	// If no children, nothing to evaluate
	if len(children) == 0 {
		return nil
	}

	// Evaluate child statuses
	allCached := true
	allSkipped := true
	anyFailed := false

	for _, child := range children {
		status := child.GetStatus()

		// Check for FAILED
		if status == apiV2beta1.PipelineTaskDetail_FAILED {
			anyFailed = true
		}

		// Check if all are CACHED
		if status != apiV2beta1.PipelineTaskDetail_CACHED {
			allCached = false
		}

		// Check if all are SKIPPED
		if status != apiV2beta1.PipelineTaskDetail_SKIPPED {
			allSkipped = false
		}
	}

	// Determine the new status for the parent
	var newStatus apiV2beta1.PipelineTaskDetail_TaskState
	if anyFailed {
		newStatus = apiV2beta1.PipelineTaskDetail_FAILED
	} else if allCached {
		newStatus = apiV2beta1.PipelineTaskDetail_CACHED
	} else if allSkipped {
		newStatus = apiV2beta1.PipelineTaskDetail_SKIPPED
	} else {
		newStatus = apiV2beta1.PipelineTaskDetail_SUCCEEDED
	}

	// Update the parent task status
	parentTask.Status = newStatus
	parentTask.EndTime = timestamppb.New(time.Now())
	_, err := kfpAPIClient.UpdateTask(ctx, &apiV2beta1.UpdateTaskRequest{
		TaskId: parentTask.GetTaskId(),
		Task:   parentTask,
	})
	if err != nil {
		return fmt.Errorf("failed to update parent task %s status to %s: %w", parentTask.GetTaskId(), newStatus, err)
	}

	return nil
}

// Execute calls executeV2, updates the cache, and creates artifacts for outputs.
func (l *LauncherV2) Execute(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to execute component: %w", err)
		}
	}()

	defer stopWaitingArtifacts(l.executorInput.GetInputs().GetArtifacts())

	// Close any open buckets in the cache
	defer func() {
		for _, bucket := range l.openedBucketCache {
			_ = bucket.Close()
		}
	}()

	// Fetch Launcher config and initialize KFP API client if not already set (testing mode)
	// Production path: fetch real config and create real client
	launcherConfig, err := config.FetchLauncherConfigMap(ctx, l.clientManager.K8sClient(), l.options.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get launcher configmap: %w", err)
	}
	l.launcherConfig = launcherConfig

	if err = l.prepareOutputFolders(l.executorInput); err != nil {
		return err
	}
	var executorOutput *pipelinespec.ExecutorOutput
	executorOutput, err = l.executeV2(ctx)
	if err != nil {
		return err
	}

	// Update task outputs for parameters
	if executorOutput != nil && len(executorOutput.GetParameterValues()) > 0 {
		params := make([]*apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter, 0, len(executorOutput.GetParameterValues()))
		for key, val := range executorOutput.GetParameterValues() {
			param := &apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
				ParameterKey: key,
				Type:         apiV2beta1.IOType_OUTPUT,
				Value:        val,
				Producer: &apiV2beta1.IOProducer{
					TaskName: l.options.TaskSpec.GetTaskInfo().GetName(),
				}}
			if l.options.IterationIndex != nil {
				param.Producer.Iteration = l.options.IterationIndex
				param.Type = apiV2beta1.IOType_ITERATOR_OUTPUT
			}
			params = append(params, param)
		}

		l.options.Task.Outputs = &apiV2beta1.PipelineTaskDetail_InputOutputs{Parameters: params}
		_, updateErr := l.clientManager.KFPAPIClient().UpdateTask(ctx, &apiV2beta1.UpdateTaskRequest{
			TaskId: l.options.Task.GetTaskId(),
			Task:   l.options.Task})
		if updateErr != nil {
			return fmt.Errorf("failed to update task outputs: %w", updateErr)
		}
	}

	l.options.Task.Status = apiV2beta1.PipelineTaskDetail_SUCCEEDED
	l.options.Task.EndTime = timestamppb.New(time.Now())

	// Update current task status to SUCCEEDED before calling updateStatuses
	// This is important because if we don't have this task updated, we will always stop at its direct immediate parent during traversal.
	_, updateErr := l.clientManager.KFPAPIClient().UpdateTask(ctx, &apiV2beta1.UpdateTaskRequest{
		TaskId: l.options.Task.GetTaskId(),
		Task:   l.options.Task})
	if updateErr != nil {
		return fmt.Errorf("failed to update task status to SUCCEEDED: %w", updateErr)
	}

	// Refresh run before updating statuses
	refreshedRun, err := l.clientManager.KFPAPIClient().GetRun(ctx, &apiV2beta1.GetRunRequest{RunId: l.options.Run.GetRunId()})
	if err != nil {
		return fmt.Errorf("failed to refresh run: %w", err)
	}
	l.options.Run = refreshedRun

	// TODO(HumairAK): Let's have API Server handle this call instead of doing it here.
	err = updateStatuses(ctx, l.clientManager.KFPAPIClient(), l.options.Run, l.options.Task)
	if err != nil {
		return fmt.Errorf("failed to update statuses: %w", err)
	}
	return nil
}

func (l *LauncherV2) Info() string {
	content, err := protojson.Marshal(l.executorInput)
	if err != nil {
		content = []byte("{}")
	}
	return strings.Join([]string{
		"launcher info:",
		fmt.Sprintf("executorInput=%s\n", prettyPrint(string(content))),
	}, "\n")
}

func (o *LauncherV2Options) validate() error {
	empty := func(s string) bool { return len(s) == 0 }
	err := func(s string) error { return fmt.Errorf("invalid launcher options: must specify %s", s) }
	if empty(o.Namespace) {
		return err("Namespace")
	}
	if empty(o.PodName) {
		return err("PodName")
	}
	if empty(o.PodUID) {
		return err("PodUID")
	}
	if o.PipelineName == "" {
		return err("PipelineName")
	}
	return nil
}

// executeV2 handles placeholder substitution for inputs, calls execute to
// execute end user logic, and uploads the resulting output Artifacts.
func (l *LauncherV2) executeV2(ctx context.Context) (*pipelinespec.ExecutorOutput, error) {
	// Add parameter default values to executorInput, if there is not already a user input.
	// This process is done in the launcher because we let the component resolve default values internally.
	// Variable executorInputWithDefault is a copy so we don't alter the original data.
	executorInputWithDefault, err := addDefaultParams(l.executorInput, l.options.ComponentSpec)
	if err != nil {
		return nil, err
	}

	// Fill in placeholders with runtime values.
	compiledCmd, compiledArgs, err := compileCmdAndArgs(executorInputWithDefault, l.command, l.args)
	if err != nil {
		return nil, err
	}

	executorOutput, err := l.execute(ctx, compiledCmd, compiledArgs)
	if err != nil {
		return nil, err
	}

	// These are not added in execute(), because execute() is shared between v2 compatible and v2 engine launcher.
	// In v2 compatible mode, we get output parameter info from runtimeInfo. In v2 engine, we get it from component spec.
	// Because of the difference, we cannot put parameter collection logic in one method.
	err = l.collectOutputParameters(executorOutput)
	if err != nil {
		return nil, err
	}

	// Upload artifacts from local disk to remote store.
	err = l.uploadOutputArtifacts(ctx, executorOutput)
	if err != nil {
		return nil, err
	}

	// Propagate outputs up the DAG hierarchy for parents that declare these outputs
	err = l.propagateOutputsUpDAG(ctx)
	if err != nil {
		return nil, err
	}

	return executorOutput, nil
}

// collectOutputParameters collect output parameters from local disk and add them
// to executor output.
func (l *LauncherV2) collectOutputParameters(executorOutput *pipelinespec.ExecutorOutput) error {
	if executorOutput.ParameterValues == nil {
		executorOutput.ParameterValues = make(map[string]*structpb.Value)
	}
	outputParameters := executorOutput.GetParameterValues()
	for name, param := range l.executorInput.GetOutputs().GetParameters() {
		_, ok := outputParameters[name]
		if ok {
			// If the output parameter was already specified in output metadata file,
			// we don't need to collect it from file, because output metadata file has
			// the highest priority.
			continue
		}
		paramSpec, ok := l.options.ComponentSpec.GetOutputDefinitions().GetParameters()[name]
		if !ok {
			return fmt.Errorf("failed to find output parameter name=%q in component spec", name)
		}
		msg := func(err error) error {
			return fmt.Errorf("failed to read output parameter name=%q type=%q path=%q: %w", name, paramSpec.GetParameterType(), param.GetOutputFile(), err)
		}
		b, err := l.fileSystem.ReadFile(param.GetOutputFile())
		if err != nil {
			return msg(err)
		}
		value, err := textToPbValue(string(b), paramSpec.GetParameterType())
		if err != nil {
			return msg(err)
		}
		outputParameters[name] = value
	}
	return nil
}

func prettyPrint(jsonStr string) string {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, []byte(jsonStr), "", "  ")
	if err != nil {
		return jsonStr
	}
	return string(prettyJSON.Bytes())
}

const OutputMetadataFilepath = "/tmp/kfp_outputs/output_metadata.json"

// We overwrite this as a DI mechanism for testing getLogWriter.
var osCreateFunc = os.Create

// getLogWriter returns an io.Writer that can either be single-channel to stdout
// or dual-channel to stdout AND a log file based on the URI of a log artifact
// in the supplied ArtifactList. Downstream, the resulting log file gets
// uploaded to the object store.
func getLogWriter(artifacts map[string]*pipelinespec.ArtifactList) (writer io.Writer) {
	logsArtifactList, ok := artifacts["executor-logs"]

	if !ok || len(logsArtifactList.Artifacts) != 1 {
		return os.Stdout
	}

	logURI := logsArtifactList.Artifacts[0].Uri
	logFilePath, err := LocalPathForURI(logURI)
	if err != nil {
		glog.Errorf("Error converting log artifact URI, %s, to file path.", logURI)
		return os.Stdout
	}

	logFile, err := osCreateFunc(logFilePath)
	if err != nil {
		glog.Errorf("Error creating logFilePath, %s.", logFilePath)
		return os.Stdout
	}

	return io.MultiWriter(os.Stdout, logFile)
}

// ExecuteForTesting is a test-only method that executes the launcher with mocked dependencies.
// It runs the full execution flow including artifact uploads but uses the provided mock dependencies.
// This method should only be used in tests.
func (l *LauncherV2) ExecuteForTesting(ctx context.Context) (*pipelinespec.ExecutorOutput, error) {
	return l.executeV2(ctx)
}

// execute downloads input artifacts, prepares the execution environment,
// executes the end user code, and returns the outputs.
func (l *LauncherV2) execute(
	ctx context.Context,
	cmd string,
	args []string,
) (*pipelinespec.ExecutorOutput, error) {
	if err := l.downloadArtifacts(ctx); err != nil {
		return nil, err
	}

	if err := l.prepareOutputFolders(l.executorInput); err != nil {
		return nil, err
	}

	var writer io.Writer
	if l.options.PublishLogs == "true" {
		writer = getLogWriter(l.executorInput.Outputs.GetArtifacts())
	} else {
		writer = os.Stdout
	}

	defer glog.Flush()

	// Execute end user code using the command executor interface.
	if err := l.cmdExecutor.Run(ctx, cmd, args, os.Stdin, writer, writer); err != nil {
		return nil, err
	}

	return l.getExecutorOutputFile(l.executorInput.GetOutputs().GetOutputFile())
}

// uploadOutputArtifacts iterates over all the Artifacts retrieved from the
// executor output and uploads them to the object store and registers them
// with the KFP API.
func (l *LauncherV2) uploadOutputArtifacts(
	ctx context.Context,
	executorOutput *pipelinespec.ExecutorOutput,
) error {
	// Manage an opened bucket cache to minimize pool
	var openedBucketCache = map[string]*blob.Bucket{}
	defer func() {
		for _, bucket := range openedBucketCache {
			_ = bucket.Close()
		}
	}()

	// After successful execution and uploads, record outputs in KFP API
	// Create artifactsMap for each output port
	artifactsMap := map[string][]*apiV2beta1.Artifact{}
	for artifactKey, artifactList := range l.executorInput.GetOutputs().GetArtifacts() {
		artifactsMap[artifactKey] = []*apiV2beta1.Artifact{}
		for _, outputArtifact := range artifactList.Artifacts {
			glog.Infof("outputArtifact in uploadOutputArtifacts call: %s", outputArtifact.Name)
			// Merge executor output artifact info with executor input
			if list, ok := executorOutput.Artifacts[artifactKey]; ok && len(list.Artifacts) > 0 {
				mergeRuntimeArtifacts(list.Artifacts[0], outputArtifact)
			}
			// OCI artifactsMap are accessed via shared storage of a Modelcar
			if strings.HasPrefix(outputArtifact.Uri, "oci://") {
				continue
			}

			artifactType, err := inferArtifactType(outputArtifact.GetType())
			if err != nil {
				return fmt.Errorf("failed to infer artifact type for port %s: %w", artifactKey, err)
			}

			// Metric artifacts don't have a URI, only a numberValue
			if artifactType == apiV2beta1.Artifact_Metric {
				// Each key/value pair in `metadata` equates to a new Artifact
				for key, value := range outputArtifact.GetMetadata().GetFields() {
					numVal, ok := value.Kind.(*structpb.Value_NumberValue)
					if !ok {
						return fmt.Errorf("metric value %q must be a number, got %T", key, value.Kind)
					}
					artifact := &apiV2beta1.Artifact{
						Name:        key,
						Description: "",
						Type:        artifactType,
						Metadata:    outputArtifact.GetMetadata().GetFields(),
						NumberValue: &numVal.NumberValue,
						CreatedAt:   timestamppb.Now(),
						Namespace:   l.options.Namespace,
					}
					artifactsMap[artifactKey] = append(artifactsMap[artifactKey], artifact)
				}
			} else {
				// In this case we can still encounter metrics of type ClassificationMetric or SlicedClassificationMetric
				// which do not have a numberValue, but nor do they have a URI, their values are stored only in metadata.
				artifact := &apiV2beta1.Artifact{
					Name:        outputArtifact.GetName(),
					Description: "",
					Type:        artifactType,
					Metadata:    outputArtifact.GetMetadata().GetFields(),
					CreatedAt:   timestamppb.Now(),
				}

				// In the Classification metric case, the metric data is stored in metadata and
				// not object store
				isNotAMetric := apiV2beta1.Artifact_ClassificationMetric != artifactType &&
					apiV2beta1.Artifact_SlicedClassificationMetric != artifactType

				// If the artifact is not a metric, upload it to the object store and store the URI in the artifact
				if isNotAMetric {
					localPath, err := LocalPathForURI(outputArtifact.Uri)
					if err != nil {
						glog.Warningf("Output Artifact %q does not have a recognized storage URI %q. Skipping uploading to remote storage.",
							artifactKey, outputArtifact.Uri)
					}
					err = l.objectStore.UploadArtifact(ctx, localPath, outputArtifact.Uri, artifactKey)
					if err != nil {
						return fmt.Errorf("failed to upload output artifact %q to remote storage URI %q: %w", artifactKey, outputArtifact.Uri, err)
					}
					artifact.Uri = util.StringPointer(outputArtifact.Uri)
				}

				artifactsMap[artifactKey] = []*apiV2beta1.Artifact{artifact}
			}
		}
	}

	// Register the Artifacts with the KFP database
	//  TODO(HumairAK): This should be done in a single API call.
	for artifactKey, artifacts := range artifactsMap {
		for _, artifact := range artifacts {
			request := &apiV2beta1.CreateArtifactRequest{
				RunId:       l.options.Run.GetRunId(),
				TaskId:      l.options.Task.GetTaskId(),
				ProducerKey: artifactKey,
				Artifact:    artifact,
				Type:        apiV2beta1.IOType_OUTPUT,
			}
			if l.options.IterationIndex != nil {
				request.IterationIndex = l.options.IterationIndex
				request.Type = apiV2beta1.IOType_ITERATOR_OUTPUT
			}
			_, err := l.clientManager.KFPAPIClient().CreateArtifact(ctx, request)
			if err != nil {
				return fmt.Errorf("failed to create artifact: %w", err)
			}
		}
	}
	return nil
}

// propagateOutputsUpDAG traverses up the DAG hierarchy and creates artifact-task entries
// for parent DAGs that declare the current task's outputs in their outputDefinitions.
// This enables output collection from child tasks (e.g., loop iterations) to parent DAGs.
func (l *LauncherV2) propagateOutputsUpDAG(ctx context.Context) error {
	// If this task has no parent, nothing to propagate
	if l.options.ParentTask == nil {
		return nil
	}

	// Refresh the current task to get the latest outputs (artifacts were just uploaded)
	currentTask, err := l.clientManager.KFPAPIClient().GetTask(ctx, &apiV2beta1.GetTaskRequest{
		TaskId: l.options.Task.GetTaskId(),
	})
	if err != nil {
		return fmt.Errorf("failed to refresh task before propagation: %w", err)
	}

	currentTaskOutputs := currentTask.GetOutputs()
	if currentTaskOutputs == nil || len(currentTaskOutputs.GetArtifacts()) == 0 {
		// No artifacts to propagate
		return nil
	}

	// Start traversing up from the immediate parent
	parentTask := l.options.ParentTask
	currentScopePath := l.options.ScopePath
	isFirstLevel := true // Track if this is first-level propagation (from producing task to immediate parent)

	for parentTask != nil {

		// Get the parent's component spec to check outputDefinitions
		// We need to get the scope path for the parent task
		parentScopePath, err := util.ScopePathFromStringPath(l.options.Run.GetPipelineSpec(), parentTask.GetScopePath())
		if err != nil {
			return fmt.Errorf("failed to get scope path for parent task %s: %w", parentTask.GetTaskId(), err)
		}

		parentComponentSpec := parentScopePath.GetLast().GetComponentSpec()
		if parentComponentSpec == nil {
			return fmt.Errorf("parent task %s has no component spec", parentTask.GetTaskId())
		}

		parentOutputDefs := parentComponentSpec.GetOutputDefinitions()
		if parentOutputDefs == nil || len(parentOutputDefs.GetArtifacts()) == 0 {
			// Parent has no output definitions, stop propagating
			break
		}

		// Track artifacts propagated to this parent (for next level)
		// Map from artifact ID to struct containing key and IOType
		type propagatedInfo struct {
			key    string
			ioType apiV2beta1.IOType
		}
		newPropagatedArtifacts := make(map[string]propagatedInfo)

		// For each artifact output from the current task, check if the parent declares it
		for _, artifactIO := range currentTaskOutputs.GetArtifacts() {
			for _, artifact := range artifactIO.GetArtifacts() {
				// Find the matching output key in parent's output definitions
				// We need to use the immediate child task name (from the parent's perspective)
				// which is the last task in currentScopePath
				childTaskName := currentScopePath.GetLast().GetTaskSpec().GetTaskInfo().GetName()

				// We need to find which parent output key corresponds to this artifact
				matchingParentKey := findMatchingParentOutputKeyForChild(
					childTaskName,
					parentComponentSpec,
					artifactIO.GetArtifactKey(),
					parentOutputDefs,
				)

				if matchingParentKey == "" {
					// This output is not declared in parent's outputDefinitions
					continue
				}

				// Determine the correct IOType
				// For the first level (immediate parent), we determine based on context
				// For subsequent levels, we inherit the type from the previous level
				var ioType apiV2beta1.IOType

				// Use isFirstLevel flag instead of checking artifactIO type,
				// because artifacts from the producing task already have Type=OUTPUT set by uploadOutputArtifacts
				if isFirstLevel {
					// First level: determine type based on parent context
					if parentTask.GetType() == apiV2beta1.PipelineTaskDetail_LOOP {
						// For loop iterations, use ITERATOR_OUTPUT
						ioType = apiV2beta1.IOType_ITERATOR_OUTPUT
					} else {
						// Check if this is a ONE_OF output by looking at the parent's output definitions
						isOneOf := false
						if parentOutputDef, exists := parentOutputDefs.GetArtifacts()[matchingParentKey]; exists {
							isOneOf = parentOutputDef.GetIsArtifactList() == false &&
								parentTask.GetType() == apiV2beta1.PipelineTaskDetail_CONDITION_BRANCH
						}

						if isOneOf {
							ioType = apiV2beta1.IOType_ONE_OF_OUTPUT
						} else {
							// Default to OUTPUT for regular DAG outputs
							ioType = apiV2beta1.IOType_OUTPUT
						}
					}
				} else {
					// Multi-level propagation: inherit the type from the previous level
					ioType = artifactIO.GetType()
				}

				// Create artifact-task entry for the parent
				artifactTask := &apiV2beta1.ArtifactTask{
					ArtifactId: artifact.GetArtifactId(),
					TaskId:     parentTask.GetTaskId(),
					RunId:      l.options.Run.GetRunId(),
					Key:        matchingParentKey,
					Type:       ioType,
					Producer: &apiV2beta1.IOProducer{
						TaskName: l.options.TaskSpec.GetTaskInfo().GetName(),
					},
				}

				if l.options.IterationIndex != nil {
					artifactTask.Producer.Iteration = l.options.IterationIndex
				}

				_, err := l.clientManager.KFPAPIClient().CreateArtifactTask(ctx, &apiV2beta1.CreateArtifactTaskRequest{
					ArtifactTask: artifactTask,
				})
				if err != nil {
					return fmt.Errorf("failed to create artifact-task for parent %s: %w", parentTask.GetTaskId(), err)
				}

				// Track this artifact for next level propagation with its IOType
				newPropagatedArtifacts[artifact.GetArtifactId()] = propagatedInfo{
					key:    matchingParentKey,
					ioType: ioType,
				}
			}
		}

		// Move up to the next parent
		if parentTask.ParentTaskId == nil || *parentTask.ParentTaskId == "" {
			break
		}

		// Get the next parent task
		nextParent, err := l.clientManager.KFPAPIClient().GetTask(ctx, &apiV2beta1.GetTaskRequest{
			TaskId: *parentTask.ParentTaskId,
		})
		if err != nil {
			return fmt.Errorf("failed to get parent task %s: %w", *parentTask.ParentTaskId, err)
		}

		// For the next level, we only want to propagate the artifacts we just added to this parent
		// Build a new currentTaskOutputs with only the newly propagated artifacts
		newTaskOutputs := &apiV2beta1.PipelineTaskDetail_InputOutputs{
			Artifacts: []*apiV2beta1.PipelineTaskDetail_InputOutputs_IOArtifact{},
		}

		for artifactID, info := range newPropagatedArtifacts {
			// Find the artifact object
			var foundArtifact *apiV2beta1.Artifact
			for _, artifactIO := range currentTaskOutputs.GetArtifacts() {
				for _, artifact := range artifactIO.GetArtifacts() {
					if artifact.GetArtifactId() == artifactID {
						foundArtifact = artifact
						break
					}
				}
				if foundArtifact != nil {
					break
				}
			}

			if foundArtifact != nil {
				newTaskOutputs.Artifacts = append(newTaskOutputs.Artifacts, &apiV2beta1.PipelineTaskDetail_InputOutputs_IOArtifact{
					ArtifactKey: info.key,
					Artifacts:   []*apiV2beta1.Artifact{foundArtifact},
					Type:        info.ioType,
				})
			}
		}

		if len(newTaskOutputs.GetArtifacts()) == 0 {
			// No more artifacts to propagate
			break
		}

		// Move to the next level
		currentTaskOutputs = newTaskOutputs
		parentTask = nextParent
		currentScopePath = parentScopePath
		isFirstLevel = false // After first iteration, we're doing multi-level propagation
	}

	return nil
}

// findMatchingParentOutputKeyForChild finds the parent output key that corresponds to the child's output.
// This is a simplified version that takes the child task name directly as a parameter.
func findMatchingParentOutputKeyForChild(
	childTaskName string,
	parentComponentSpec *pipelinespec.ComponentSpec,
	childOutputKey string,
	parentOutputDefs *pipelinespec.ComponentOutputsSpec,
) string {
	// Get the task spec from the parent's perspective
	if parentComponentSpec == nil || parentComponentSpec.GetDag() == nil {
		return ""
	}

	// Look through parent's DAG tasks to find the child task
	for _, dagTask := range parentComponentSpec.GetDag().GetTasks() {
		if dagTask.GetTaskInfo().GetName() != childTaskName {
			continue
		}

		// Found the child task in parent's DAG
		// Check the task's output selectors
		if dagTask.GetComponentRef() != nil {
			// Look at the parent's output definitions to find which one uses this task's output
			for parentOutputKey := range parentOutputDefs.GetArtifacts() {
				// Check if this parent output is sourced from the child task
				// The parent output may be directly from task output or from an artifact selector
				if artifactSelectorMatches(parentComponentSpec, parentOutputKey, childTaskName, childOutputKey) {
					return parentOutputKey
				}
			}
		}
	}

	return ""
}

// artifactSelectorMatches checks if a parent output artifact selector matches the child task output
func artifactSelectorMatches(
	parentComponentSpec *pipelinespec.ComponentSpec,
	parentOutputKey string,
	childTaskName string,
	childOutputKey string,
) bool {
	// Check artifact selectors
	dag := parentComponentSpec.GetDag()
	if dag == nil || dag.GetOutputs() == nil {
		return false
	}

	artifactSelectors := dag.GetOutputs().GetArtifacts()
	if artifactSelectors == nil {
		return false
	}

	selector, exists := artifactSelectors[parentOutputKey]
	if !exists {
		return false
	}

	// Check if the selector references the child task
	for _, artifactSelector := range selector.GetArtifactSelectors() {
		if artifactSelector.GetProducerSubtask() == childTaskName &&
			artifactSelector.GetOutputArtifactKey() == childOutputKey {
			return true
		}
	}

	return false
}

// waitForModelcar assumes the Modelcar has already been validated by the init container on the launcher
// pod. This waits for the Modelcar as a sidecar container to be ready.
func waitForModelcar(artifactURI string, localPath string) error {
	glog.Infof("Waiting for the Modelcar %s to be available", artifactURI)

	for {
		_, err := os.Stat(localPath)
		if err == nil {
			glog.Infof("The Modelcar is now available at %s", localPath)

			return nil
		}

		if !os.IsNotExist(err) {
			return fmt.Errorf(
				"failed to see if the artifact %s was ready at %s; ensure the main container and Modelcar "+
					"container have the same UID (can be set with the PIPELINE_RUN_AS_USER environment variable on "+
					"the API server): %v",
				artifactURI, localPath, err)
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func (l *LauncherV2) downloadArtifacts(ctx context.Context) error {
	for artifactKey, artifactList := range l.executorInput.GetInputs().GetArtifacts() {
		for _, artifact := range artifactList.Artifacts {
			localPath, err := LocalPathForURI(artifact.Uri)
			if err != nil {
				glog.Warningf("Input Artifact %q does not have a recognized storage URI %q. Skipping downloading to local path.", artifactKey, artifact.Uri)
				continue
			}
			// OCI artifacts are accessed via shared storage of a Modelcar
			if strings.HasPrefix(artifact.Uri, "oci://") {
				err := waitForModelcar(artifact.Uri, localPath)
				if err != nil {
					return err
				}
				continue
			}
			err = l.objectStore.DownloadArtifact(ctx, artifact.Uri, localPath, artifactKey)
			if err != nil {
				return fmt.Errorf("failed to download input artifact %q from remote storage URI %q: %w", artifactKey, artifact.Uri, err)
			}
		}
	}
	return nil
}

func compileCmdAndArgs(executorInput *pipelinespec.ExecutorInput, cmd string, args []string) (string, []string, error) {
	placeholders, err := getPlaceholders(executorInput)

	executorInputJSON, err := protojson.Marshal(executorInput)
	if err != nil {
		return "", nil, fmt.Errorf("failed to convert ExecutorInput into JSON: %w", err)
	}
	executorInputJSONKey := "{{$}}"
	executorInputJSONString := string(executorInputJSON)

	compiledCmd := strings.ReplaceAll(cmd, executorInputJSONKey, executorInputJSONString)
	compiledArgs := make([]string, 0, len(args))
	for placeholder, replacement := range placeholders {
		cmd = strings.ReplaceAll(cmd, placeholder, replacement)
	}
	for _, arg := range args {
		compiledArgTemplate := strings.ReplaceAll(arg, executorInputJSONKey, executorInputJSONString)
		for placeholder, replacement := range placeholders {
			compiledArgTemplate = strings.ReplaceAll(compiledArgTemplate, placeholder, replacement)
		}
		compiledArgs = append(compiledArgs, compiledArgTemplate)
	}
	return compiledCmd, compiledArgs, nil
}

// Add executor input placeholders to provided map.
func getPlaceholders(executorInput *pipelinespec.ExecutorInput) (placeholders map[string]string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to get placeholders: %w", err)
		}
	}()
	placeholders = make(map[string]string)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ExecutorInput into JSON: %w", err)
	}

	// Read input artifact metadata.
	for name, artifactList := range executorInput.GetInputs().GetArtifacts() {
		if len(artifactList.Artifacts) == 0 {
			continue
		}
		inputArtifact := artifactList.Artifacts[0]

		// Prepare input uri placeholder.
		key := fmt.Sprintf(`{{$.inputs.artifacts['%s'].uri}}`, name)
		placeholders[key] = inputArtifact.Uri

		localPath, err := LocalPathForURI(inputArtifact.Uri)
		if err != nil {
			// Input Artifact does not have a recognized storage URI
			continue
		}

		// Prepare input path placeholder.
		key = fmt.Sprintf(`{{$.inputs.artifacts['%s'].path}}`, name)
		placeholders[key] = localPath
	}

	// Prepare output artifact placeholders.
	for name, artifactList := range executorInput.GetOutputs().GetArtifacts() {
		if len(artifactList.Artifacts) == 0 {
			continue
		}
		outputArtifact := artifactList.Artifacts[0]
		placeholders[fmt.Sprintf(`{{$.outputs.artifacts['%s'].uri}}`, name)] = outputArtifact.Uri

		localPath, err := LocalPathForURI(outputArtifact.Uri)
		if err != nil {
			return nil, fmt.Errorf("resolve output artifact %q's local path: %w", name, err)
		}
		placeholders[fmt.Sprintf(`{{$.outputs.artifacts['%s'].path}}`, name)] = localPath
	}

	// Prepare input parameter placeholders.
	for name, parameter := range executorInput.GetInputs().GetParameterValues() {
		key := fmt.Sprintf(`{{$.inputs.parameters['%s']}}`, name)
		switch t := parameter.Kind.(type) {
		case *structpb.Value_StringValue:
			placeholders[key] = parameter.GetStringValue()
		case *structpb.Value_NumberValue:
			placeholders[key] = strconv.FormatFloat(parameter.GetNumberValue(), 'f', -1, 64)
		case *structpb.Value_BoolValue:
			placeholders[key] = strconv.FormatBool(parameter.GetBoolValue())
		case *structpb.Value_ListValue:
			b, err := json.Marshal(parameter.GetListValue())
			if err != nil {
				return nil, fmt.Errorf("failed to JSON-marshal list input parameter %q: %w", name, err)
			}
			placeholders[key] = string(b)
		case *structpb.Value_StructValue:
			b, err := json.Marshal(parameter.GetStructValue())
			if err != nil {
				return nil, fmt.Errorf("failed to JSON-marshal dict input parameter %q: %w", name, err)
			}
			placeholders[key] = string(b)
		default:
			return nil, fmt.Errorf("unknown PipelineSpec Value type %T", t)
		}
	}

	// Prepare output parameter placeholders.
	for name, parameter := range executorInput.GetOutputs().GetParameters() {
		key := fmt.Sprintf(`{{$.outputs.parameters['%s'].output_file}}`, name)
		placeholders[key] = parameter.OutputFile
	}

	return placeholders, nil
}

func getArtifactSchemaType(schema *pipelinespec.ArtifactTypeSchema) (string, error) {
	switch t := schema.Kind.(type) {
	case *pipelinespec.ArtifactTypeSchema_InstanceSchema:
		return t.InstanceSchema, nil
	case *pipelinespec.ArtifactTypeSchema_SchemaTitle:
		return t.SchemaTitle, nil
	case *pipelinespec.ArtifactTypeSchema_SchemaUri:
		return "", fmt.Errorf("SchemaUri is unsupported")
	default:
		return "", fmt.Errorf("unknown type %T in ArtifactTypeSchema %+v", t, schema)
	}
}

func mergeRuntimeArtifacts(src, dst *pipelinespec.RuntimeArtifact) {
	if len(src.Uri) > 0 {
		dst.Uri = src.Uri
	}

	if src.Metadata != nil {
		if dst.Metadata == nil {
			dst.Metadata = src.Metadata
		} else {
			for k, v := range src.Metadata.Fields {
				dst.Metadata.Fields[k] = v
			}
		}
	}
}

func (l *LauncherV2) getExecutorOutputFile(path string) (*pipelinespec.ExecutorOutput, error) {
	// collect user executor output file
	executorOutput := &pipelinespec.ExecutorOutput{
		ParameterValues: map[string]*structpb.Value{},
		Artifacts:       map[string]*pipelinespec.ArtifactList{},
	}

	_, err := l.fileSystem.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			glog.Infof("output metadata file does not exist in %s", path)
			// If file doesn't exist, return an empty ExecutorOutput.
			return executorOutput, nil
		} else {
			return nil, fmt.Errorf("failed to stat output metadata file %q: %w", path, err)
		}
	}

	b, err := l.fileSystem.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read output metadata file %q: %w", path, err)
	}
	glog.Infof("ExecutorOutput: %s", prettyPrint(string(b)))

	if err := protojson.Unmarshal(b, executorOutput); err != nil {
		return nil, fmt.Errorf("failed to unmarshall ExecutorOutput in file %q: %w", path, err)
	}

	return executorOutput, nil
}

func LocalPathForURI(uri string) (string, error) {
	if strings.HasPrefix(uri, "gs://") {
		return "/gcs/" + strings.TrimPrefix(uri, "gs://"), nil
	}
	if strings.HasPrefix(uri, "minio://") {
		return "/minio/" + strings.TrimPrefix(uri, "minio://"), nil
	}
	if strings.HasPrefix(uri, "s3://") {
		return "/s3/" + strings.TrimPrefix(uri, "s3://"), nil
	}
	if strings.HasPrefix(uri, "oci://") {
		return "/oci/" + strings.ReplaceAll(strings.TrimPrefix(uri, "oci://"), "/", "_") + "/models", nil
	}
	return "", fmt.Errorf("failed to generate local path for URI %s: unsupported storage scheme", uri)
}

func (l *LauncherV2) prepareOutputFolders(executorInput *pipelinespec.ExecutorInput) error {
	for name, parameter := range executorInput.GetOutputs().GetParameters() {
		dir := filepath.Dir(parameter.OutputFile)
		if err := l.fileSystem.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %q for output parameter %q: %w", dir, name, err)
		}
	}

	for name, artifactList := range executorInput.GetOutputs().GetArtifacts() {
		if len(artifactList.Artifacts) == 0 {
			continue
		}

		for _, outputArtifact := range artifactList.Artifacts {

			localPath, err := LocalPathForURI(outputArtifact.Uri)
			if err != nil {
				return fmt.Errorf("failed to generate local storage path for output artifact %q: %w", name, err)
			}

			if err := l.fileSystem.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
				return fmt.Errorf("unable to create directory %q for output artifact %q: %w", filepath.Dir(localPath), name, err)
			}
		}
	}

	return nil
}

// Adds default parameter values if there is no user provided value
func addDefaultParams(
	executorInput *pipelinespec.ExecutorInput,
	component *pipelinespec.ComponentSpec,
) (*pipelinespec.ExecutorInput, error) {
	// Make a deep copy so we don't alter the original data
	executorInputWithDefaultMsg := proto.Clone(executorInput)
	executorInputWithDefault, ok := executorInputWithDefaultMsg.(*pipelinespec.ExecutorInput)
	if !ok {
		return nil, fmt.Errorf("bug: cloned executor input message does not have expected type")
	}

	if executorInputWithDefault.GetInputs().GetParameterValues() == nil {
		executorInputWithDefault.Inputs.ParameterValues = make(map[string]*structpb.Value)
	}
	for name, value := range component.GetInputDefinitions().GetParameters() {
		_, hasInput := executorInputWithDefault.GetInputs().GetParameterValues()[name]
		if value.GetDefaultValue() != nil && !hasInput {
			executorInputWithDefault.GetInputs().GetParameterValues()[name] = value.GetDefaultValue()
		}
	}
	return executorInputWithDefault, nil
}
