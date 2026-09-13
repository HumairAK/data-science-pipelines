// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type namespaceOnlyArtifactSAR struct {
	allowedNamespace string
}

func (s namespaceOnlyArtifactSAR) Create(_ context.Context, review *authorizationv1.SubjectAccessReview, _ metav1.CreateOptions) (*authorizationv1.SubjectAccessReview, error) {
	allowed := review.Spec.ResourceAttributes != nil && review.Spec.ResourceAttributes.Namespace == s.allowedNamespace
	return &authorizationv1.SubjectAccessReview{Status: authorizationv1.SubjectAccessReviewStatus{Allowed: allowed}}, nil
}

func TestMigratedArtifactIsListableReadableAndDownloadable(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	manager := resource.NewFakeClientManagerOrFatalV2()
	defer manager.Close()
	resourceManager := resource.NewResourceManager(manager, &resource.ResourceManagerOptions{CollectMetrics: false})
	run := &model.Run{UUID: "migrated-run", K8SName: "migrated-run", DisplayName: "migrated-run", Namespace: "team-a", StorageState: model.StorageStateAvailable, RunDetails: model.RunDetails{CreatedAtInSec: 1, State: model.RuntimeStateSucceeded}}
	_, err := manager.RunStore().CreateRun(run)
	require.NoError(t, err)
	task, err := manager.TaskStore().CreateTask(&model.Task{Namespace: "team-a", RunUUID: run.UUID, Name: "producer", State: model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED)})
	require.NoError(t, err)
	uri := "minio://mlpipeline/v2/artifacts/shared/output"
	artifact, err := manager.ArtifactStore().CreateArtifact(&model.Artifact{Namespace: "team-a", Type: model.ArtifactType(apiv2beta1.Artifact_Model), URI: &uri, Name: "historical-output"})
	require.NoError(t, err)
	_, err = manager.ArtifactTaskStore().CreateArtifactTask(&model.ArtifactTask{ArtifactID: artifact.UUID, TaskID: task.UUID, Type: model.IOType(apiv2beta1.IOType_OUTPUT), RunUUID: run.UUID, ArtifactKey: "output"})
	require.NoError(t, err)

	artifactServer := createArtifactServer(resourceManager)
	listed, err := artifactServer.ListArtifacts(ctxWithUser(), &apiv2beta1.ListArtifactRequest{Namespace: "team-a", PageSize: 20})
	require.NoError(t, err)
	require.Len(t, listed.GetArtifacts(), 1)
	require.Equal(t, artifact.UUID, listed.GetArtifacts()[0].GetArtifactId())
	fetched, err := artifactServer.GetArtifact(ctxWithUser(), &apiv2beta1.GetArtifactRequest{ArtifactId: artifact.UUID})
	require.NoError(t, err)
	require.Equal(t, uri, fetched.GetUri())
	require.Equal(t, "team-a", fetched.GetNamespace())

	// The download endpoint resolves the historical run/task artifact through
	// the native run manifest and streams the existing object-store payload.
	filePath := "migrated-run/producer/output"
	require.NoError(t, resourceManager.ObjectStore().AddFile(context.Background(), []byte("historical payload"), filePath))
	workflow := createWorkflowWithArtifact(run.UUID, task.Name, "output", filePath)
	run.WorkflowRuntimeManifest = model.LargeText(workflow.ToStringForStore())
	require.NoError(t, manager.RunStore().UpdateRun(run))
	req := httptest.NewRequest(http.MethodGet, "/apis/v2beta1/runs/migrated-run/nodes/producer/artifacts/output:read", nil).WithContext(ctxWithUser())
	req = mux.SetURLVars(req, map[string]string{RunKey: run.UUID, NodeKey: task.Name, ArtifactNameKey: "output"})
	recorder := httptest.NewRecorder()
	NewRunArtifactServer(resourceManager).ReadArtifact(recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code)
	var body map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	decoded, err := base64.StdEncoding.DecodeString(body["data"])
	require.NoError(t, err)
	require.Equal(t, "historical payload", string(decoded))
}

func TestMigratedArtifactCrossNamespaceAccessIsDenied(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	manager := resource.NewFakeClientManagerOrFatalV2()
	defer manager.Close()
	resourceManager := resource.NewResourceManager(manager, &resource.ResourceManagerOptions{CollectMetrics: false})
	uri := "minio://mlpipeline/v2/artifacts/shared/team-b-output"
	artifact, err := manager.ArtifactStore().CreateArtifact(&model.Artifact{Namespace: "team-b", Type: model.ArtifactType(apiv2beta1.Artifact_Model), URI: &uri, Name: "team-b-output"})
	require.NoError(t, err)
	manager.SubjectAccessReviewClientFake = namespaceOnlyArtifactSAR{allowedNamespace: "team-a"}
	resourceManager = resource.NewResourceManager(manager, &resource.ResourceManagerOptions{CollectMetrics: false})

	_, err = createArtifactServer(resourceManager).GetArtifact(ctxWithUser(), &apiv2beta1.GetArtifactRequest{ArtifactId: artifact.UUID})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not authorized")
}

func TestMigratedMetricIsRetrievableThroughRunAPI(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	manager, resourceManager, run := initWithOneTimeRunV2(t)
	defer manager.Close()
	require.NoError(t, manager.RunStore().CreateV1Metric(&model.RunMetricV1{RunUUID: run.UUID, NodeID: "metric-task", Name: "accuracy", NumberValue: 0.91, Format: "RAW", Payload: model.LargeText(`{"source":"mlmd"}`)}))

	response, err := NewRunServerV1(resourceManager, &RunServerOptions{CollectMetrics: false}).GetRunV1(ctxWithUser(), &apiv1beta1.GetRunRequest{RunId: run.UUID})
	require.NoError(t, err)
	require.Len(t, response.GetRun().GetMetrics(), 1)
	require.Equal(t, "accuracy", response.GetRun().GetMetrics()[0].GetName())
	numberValue, ok := response.GetRun().GetMetrics()[0].GetValue().(*apiv1beta1.RunMetric_NumberValue)
	require.True(t, ok)
	require.InDelta(t, 0.91, numberValue.NumberValue, 0.0001)
}
