package component

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/golang/glog"
)

type ImportLauncher struct {
	opts          LauncherV2Options
	clientManager client_manager.ClientManagerInterface
}

func NewImporterLauncher(
	launcherV2Opts *LauncherV2Options,
	clientManager client_manager.ClientManagerInterface,
) (l *ImportLauncher, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to create importer launcher: %w", err)
		}
	}()
	err = launcherV2Opts.validate()
	if err != nil {
		return nil, err
	}
	return &ImportLauncher{
		opts:          *launcherV2Opts,
		clientManager: clientManager,
	}, nil
}

func (l *ImportLauncher) Execute(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to execute importer component: %w", err)
		}
	}()
	driverAPI := l.clientManager.DriverAPI()
	parentTaskID := l.opts.ParentTask.GetTaskId()
	createdTask, err := driverAPI.CreateTask(ctx, &apiV2beta1.CreateTaskRequest{
		Task: &apiV2beta1.PipelineTaskDetail{
			Name:         l.opts.TaskSpec.GetTaskInfo().GetName(),
			DisplayName:  l.opts.TaskSpec.GetTaskInfo().GetName(),
			RunId:        l.opts.Run.RunId,
			ParentTaskId: &parentTaskID,
			Type:         apiV2beta1.PipelineTaskDetail_IMPORTER,
			Status:       apiV2beta1.PipelineTaskDetail_RUNNING,
			ScopePath:    l.opts.ScopePath.StringPath(),
			StartTime:    timestamppb.Now(),
			CreateTime:   timestamppb.Now(),
			Pods: []*apiV2beta1.PipelineTaskDetail_TaskPod{
				{
					Name: l.opts.PodName,
					Uid:  l.opts.PodUID,
					Type: apiV2beta1.PipelineTaskDetail_EXECUTOR,
				},
			},
		},
	})
	if err != nil {
		return err
	}
	if createdTask == nil {
		return fmt.Errorf("failed to create task for importer execution")
	}
	l.opts.Task = createdTask

	artifact, err := l.findOrNewArtifactToImport(ctx)
	if err != nil {
		return err
	}
	artifactOutputKey, err := l.getArtifactOutputKey()
	if err != nil {
		return err
	}

	if createdTask.Outputs == nil {
		createdTask.Outputs = &apiV2beta1.PipelineTaskDetail_InputOutputs{
			Artifacts: make([]*apiV2beta1.PipelineTaskDetail_InputOutputs_IOArtifact, 0),
		}
	} else if createdTask.Outputs.Artifacts == nil {
		createdTask.Outputs.Artifacts = make([]*apiV2beta1.PipelineTaskDetail_InputOutputs_IOArtifact, 0)
	}
	createdTask.Outputs.Artifacts = append(createdTask.Outputs.Artifacts, &apiV2beta1.PipelineTaskDetail_InputOutputs_IOArtifact{
		Artifacts:   []*apiV2beta1.Artifact{artifact},
		Type:        apiV2beta1.IOType_OUTPUT,
		ArtifactKey: artifactOutputKey,
		Producer: &apiV2beta1.IOProducer{
			TaskName: l.opts.TaskSpec.GetTaskInfo().GetName(),
		},
	})
	createdTask.Status = apiV2beta1.PipelineTaskDetail_SUCCEEDED
	createdTask.EndTime = timestamppb.Now()
	_, err = driverAPI.UpdateTask(ctx, &apiV2beta1.UpdateTaskRequest{
		TaskId: createdTask.TaskId,
		Task:   createdTask,
	})
	return nil
}

// findOrNewArtifactToImport will find an artifact to import.
// If Re-Import on the importer spec is true then a new artifact is returned for creation.
// If Re-Import is False, then we search for a matching artifact, if:
//   - A match is found, then we return the match
//   - No match is found, then a new artifact is returned for creation.
func (l *ImportLauncher) findOrNewArtifactToImport(ctx context.Context) (artifact *apiV2beta1.Artifact, err error) {
	artifactToImport, err := l.ImportSpecToArtifact()
	if err != nil {
		return nil, err
	}
	if l.opts.ImporterSpec.Reimport {
		return artifactToImport, nil
	}
	matchedArtifact, err := l.findMatchedArtifact(ctx, artifactToImport)
	if err != nil {
		return nil, err
	}
	if matchedArtifact != nil {
		return matchedArtifact, nil
	}
	return artifactToImport, nil
}

func (l *ImportLauncher) findMatchedArtifact(ctx context.Context, artifactToMatch *apiV2beta1.Artifact) (matchedArtifact *apiV2beta1.Artifact, err error) {
	artifacts, err := l.clientManager.DriverAPI().ListArtifactsByURI(ctx, artifactToMatch.GetUri(), l.opts.Namespace)
	if err != nil {
		return nil, err
	}
	for _, artifact := range artifacts {
		if artifact.GetUri() == artifactToMatch.GetUri() {
			return artifact, nil
		}
	}
	for _, candidateArtifact := range artifacts {
		if artifactsAreEqual(artifactToMatch, candidateArtifact) {
			return candidateArtifact, nil
		}
	}
	// No match found
	return nil, nil
}

func artifactsAreEqual(artifact1, artifact2 *apiV2beta1.Artifact) bool {
	if artifact1.GetType() != artifact2.GetType() {
		return false
	}
	if artifact1.GetUri() != artifact2.GetUri() {
		return false
	}
	if artifact1.GetName() != artifact2.GetName() {
		return false
	}
	if artifact1.GetDescription() != artifact2.GetDescription() {
		return false
	}
	// Compare metadata fields
	metadata1 := artifact1.GetMetadata()
	metadata2 := artifact2.GetMetadata()
	if len(metadata1) != len(metadata2) {
		return false
	}
	for k, v1 := range metadata1 {
		if v2, exists := metadata2[k]; !exists || v1 != v2 {
			return false
		}
	}
	return true
}

func (l *ImportLauncher) ImportSpecToArtifact() (artifact *apiV2beta1.Artifact, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to create Artifact from ImporterSpec: %w", err)
		}
	}()

	importerSpec := l.opts.ImporterSpec
	artifactType, err := inferArtifactType(importerSpec.GetTypeSchema())
	if err != nil {
		return nil, fmt.Errorf("failed to get schemaType from importer spec: %w", err)
	}
	// Resolve artifact URI. Can be one of two sources:
	// 1) Constant
	// 2) Runtime Parameter
	// TODO(Humair): The logic here is very similar to how InputParameters are resolved in the driver's resolver package.
	// We should consolidate this logic.
	var artifactUri string
	if importerSpec.GetArtifactUri().GetConstant() != nil {
		glog.Infof("Artifact URI as constant: %+v", importerSpec.GetArtifactUri().GetConstant())
		artifactUri = importerSpec.GetArtifactUri().GetConstant().GetStringValue()
		if artifactUri == "" {
			return nil, fmt.Errorf("empty Artifact URI constant value")
		}
	} else if importerSpec.GetArtifactUri().GetRuntimeParameter() != "" {
		paramName := importerSpec.GetArtifactUri().GetRuntimeParameter()
		taskInput, ok := l.opts.TaskSpec.GetInputs().GetParameters()[paramName]
		if !ok {
			return nil, fmt.Errorf("cannot find parameter %s in task input to fetch artifact uri", paramName)
		}
		componentInput := taskInput.GetComponentInputParameter()
		var ioParam *apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter
		for _, inputParam := range l.opts.ParentTask.GetInputs().GetParameters() {
			if inputParam.ParameterKey == componentInput {
				ioParam = inputParam
				break
			}
		}
		if ioParam == nil {
			return nil, fmt.Errorf("cannot find parameter %s in parent task input to fetch artifact uri", componentInput)
		}
		artifactUri = ioParam.GetValue().GetStringValue()
		if artifactUri == "" {
			return nil, fmt.Errorf("empty artifact URI runtime value for parameter %s", paramName)
		}
	} else {
		return nil, fmt.Errorf("artifact uri not provided")
	}

	// TODO(HumairAK): Allow user to specify a canonical artifact Name & Description when importing
	// For now we infer the name from the URI object name.
	artifactName, err := inferArtifactName(artifactUri)
	if err != nil {
		return nil, fmt.Errorf("failed to extract filename from artifact uri: %w", err)
	}
	artifact = &apiV2beta1.Artifact{
		Type:        artifactType,
		Uri:         &artifactUri,
		Name:        artifactName,
		Description: "",
	}
	if importerSpec.Metadata != nil {
		artifact.Metadata = importerSpec.Metadata.GetFields()
	}
	if strings.HasPrefix(artifactUri, "oci://") {
		if artifactType != apiV2beta1.Artifact_Model {
			return nil, fmt.Errorf("the %s artifact type does not support OCI registries", apiV2beta1.Artifact_Model)
		}
		return artifact, nil
	}
	return artifact, nil
}

func (l *ImportLauncher) getArtifactOutputKey() (string, error) {
	outputNames := make([]string, 0, len(l.opts.ComponentSpec.GetOutputDefinitions().GetArtifacts()))
	for name := range l.opts.ComponentSpec.GetOutputDefinitions().GetArtifacts() {
		outputNames = append(outputNames, name)
	}
	if len(outputNames) != 1 {
		return "", fmt.Errorf("failed to extract output artifact name from componentOutputSpec")
	}
	return outputNames[0], nil
}

func inferArtifactType(typeSchema *pipelinespec.ArtifactTypeSchema) (apiV2beta1.Artifact_ArtifactType, error) {
	schemaType, err := getArtifactSchemaType(typeSchema)
	if err != nil {
		return apiV2beta1.Artifact_TYPE_UNSPECIFIED, fmt.Errorf("failed to get schemaType from importer spec: %w", err)
	}
	return artifactTypeSchemaToArtifactType(schemaType)
}

func inferArtifactName(uri string) (string, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("invalid URI: %w", err)
	}
	// For cases like "s3://bucket/path/to/file.txt"
	if parsed.Scheme != "" && parsed.Host != "" {
		return path.Base(parsed.Path), nil
	}
	// For "https://minio.local/bucket/path/to/file.txt"
	if parsed.Scheme != "" && parsed.Host == "" {
		return path.Base(parsed.Path), nil
	}
	// For URLs without a scheme, e.g. "bucket/path/to/file.txt"
	cleaned := strings.TrimSuffix(uri, "/")
	return path.Base(cleaned), nil
}
