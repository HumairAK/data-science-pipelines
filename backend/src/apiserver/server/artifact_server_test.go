package server

import (
	"context"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

func createArtifactServer(resourceManager *resource.ResourceManager) *ArtifactServer {
	return &ArtifactServer{resourceManager: resourceManager}
}

// ctxWithUser returns a context with a fake user identity header so that
// authorization in multi-user mode passes in tests.
func ctxWithUser() context.Context {
	header := common.GetKubeflowUserIDHeader()
	prefix := common.GetKubeflowUserIDPrefix()
	// Typical header value is like: "accounts.google.com:alice@example.com"
	val := prefix + "test-user@example.com"
	md := metadata.New(map[string]string{header: val})
	return metadata.NewIncomingContext(context.Background(), md)
}

func TestArtifactServer_CreateArtifact_MultiUserCreateAndGet_Succeeds(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	req := &apiv2beta1.CreateArtifactRequest{Artifact: &apiv2beta1.Artifact{
		Namespace: "ns1",
		Type:      apiv2beta1.Artifact_Model,
		Uri:       "gs://b/f",
		Name:      "a1",
	}}
	created, err := s.CreateArtifact(ctxWithUser(), req)
	assert.NoError(t, err)
	assert.NotEmpty(t, created.GetArtifactId())
	assert.Equal(t, "ns1", created.GetNamespace())
	assert.Equal(t, apiv2beta1.Artifact_Model, created.GetType())
	assert.Equal(t, "gs://b/f", created.GetUri())
	assert.Equal(t, "a1", created.GetName())
}

func TestArtifactServer_UpdateArtifact_HappyPath(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	created, err := s.CreateArtifact(ctxWithUser(), &apiv2beta1.CreateArtifactRequest{
		Artifact: &apiv2beta1.Artifact{
			Namespace: "ns1",
			Type:      apiv2beta1.Artifact_Model,
			Uri:       "gs://b/f",
			Name:      "a1",
		},
	})
	assert.NoError(t, err)
	created.Name = "a1-upd"
	created.Type = apiv2beta1.Artifact_Dataset
	upd, err := s.UpdateArtifact(ctxWithUser(), &apiv2beta1.UpdateArtifactRequest{Artifact: created})
	assert.NoError(t, err)
	assert.Equal(t, "a1-upd", upd.GetName())
	assert.Equal(t, apiv2beta1.Artifact_Dataset, upd.GetType())
}

func TestArtifactServer_ListArtifacts_HappyPath(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	_, err := s.CreateArtifact(ctxWithUser(), &apiv2beta1.CreateArtifactRequest{Artifact: &apiv2beta1.Artifact{
		Namespace: "ns1",
		Type:      apiv2beta1.Artifact_Model,
		Uri:       "gs://b/f",
		Name:      "a1",
	}})
	listResp, err := s.ListArtifacts(ctxWithUser(), &apiv2beta1.ListArtifactRequest{
		Namespace: "ns1",
		PageSize:  10,
	})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, int(listResp.GetTotalSize()), 1)
	assert.GreaterOrEqual(t, len(listResp.GetArtifacts()), 1)
}

func TestArtifactServer_GetArtifact_Errors(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Missing ID
	_, err := s.GetArtifact(context.Background(), &apiv2beta1.GetArtifactRequest{ArtifactId: ""})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())

	// Non-existent
	_, err = s.GetArtifact(context.Background(), &apiv2beta1.GetArtifactRequest{ArtifactId: "does-not-exist"})
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestArtifactServer_LogAndGetMetrics(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Seed a run and a task via RunStore/TaskStore APIs exposed by resource manager
	// Create a task directly in storage (no need to create a full run CR for tests)
	task, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace:    "ns1",
		PipelineName: "p",
		RunUUID:      "run-1",
		Name:         "t1",
		Status:       1,
	})
	assert.NoError(t, err)

	// Log metric for the task
	logMetricRequest := &apiv2beta1.LogMetricRequest{
		Metric: &apiv2beta1.Metric{
			RunId:  "run-1",
			TaskId: task.UUID,
			Name:   "accuracy",
			Value:  &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: 0.2}},
		},
	}
	metric, err := s.LogMetric(context.Background(), logMetricRequest)
	assert.NoError(t, err)
	assert.Equal(t, "accuracy", metric.GetName())

	// Get metric
	gmet, err := s.GetMetric(context.Background(), &apiv2beta1.GetMetricRequest{TaskId: task.UUID, Name: "accuracy"})
	assert.NoError(t, err)
	assert.Equal(t, "accuracy", gmet.GetName())

	// List metrics
	lmet, err := s.ListMetrics(context.Background(),
		&apiv2beta1.ListMetricsRequest{
			Namespace: "ns1",
			TaskIds:   []string{task.UUID},
			PageSize:  10,
		})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), lmet.GetTotalSize())
	assert.Equal(t, 1, len(lmet.GetMetrics()))
}

func TestArtifactServer_ValidationErrors(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Create: missing artifact
	_, err := s.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())

	// Create: missing namespace
	_, err = s.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{Artifact: &apiv2beta1.Artifact{}})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())

	// Update: missing id
	_, err = s.UpdateArtifact(context.Background(), &apiv2beta1.UpdateArtifactRequest{Artifact: &apiv2beta1.Artifact{}})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())

	// LogMetric: missing fields
	_, err = s.LogMetric(context.Background(), &apiv2beta1.LogMetricRequest{Metric: &apiv2beta1.Metric{}})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
}

func TestArtifactServer_Authorization_MultiUser(t *testing.T) {
	// Turn on MU mode by setting viper flag
	// Note: IsMultiUserMode() reads from viper, so configure it here
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	// In MU, Create should preserve namespace; and List with empty namespace should fail
	s := createArtifactServer(resourceManager)

	// By default FakeResourceManager authorizes everything in MU, unless namespace is empty
	// ListArtifacts with empty namespace should fail in MU
	_, err := s.ListArtifacts(ctxWithUser(), &apiv2beta1.ListArtifactRequest{Namespace: ""})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
}

func TestArtifactServer_SingleUserNamespaceEmpty(t *testing.T) {
	// Ensure single-user mode
	viper.Set(common.MultiUserMode, "false")
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Even if request carries a namespace, in single-user mode it should be cleared/empty in stored artifact
	created, err := s.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{
		Artifact: &apiv2beta1.Artifact{
			Namespace: "ns1",
			Type:      apiv2beta1.Artifact_Artifact,
			Uri:       "u",
			Name:      "a",
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "", created.GetNamespace())
}

const (
	serverRunID1 = "run-1"
	serverRunID2 = "run-2"
)

// seedArtifactTasks sets up two runs, two tasks, two artifacts and three links.
// Returns server, clientManager, entities.
func seedArtifactTasks(t *testing.T) (*ArtifactServer, *resource.FakeClientManager, *model.Task, *model.Task, *model.Artifact, *model.Artifact) {
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })
	clientManager := resource.NewFakeClientManagerOrFatalV2()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Runs
	_, err := clientManager.RunStore().CreateRun(&model.Run{
		UUID:         serverRunID1,
		ExperimentId: "",
		K8SName:      "r1",
		DisplayName:  "r1",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns1",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			State:            model.RuntimeStateRunning,
		},
	})
	assert.NoError(t, err)
	_, err = clientManager.RunStore().CreateRun(&model.Run{
		UUID:         serverRunID2,
		ExperimentId: "",
		K8SName:      "r2",
		DisplayName:  "r2",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns1",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			State:            model.RuntimeStateRunning,
		},
	})

	// Tasks
	t1, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace:    "ns1",
		PipelineName: "p1",
		RunUUID:      serverRunID1,
		Name:         "t1",
		Status:       1,
	})
	assert.NoError(t, err)
	t2, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace:    "ns1",
		PipelineName: "p1",
		RunUUID:      serverRunID2,
		Name:         "t2",
		Status:       1,
	})
	assert.NoError(t, err)

	// Artifacts
	art1, err := clientManager.ArtifactStore().CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      int32(apiv2beta1.Artifact_Artifact),
		Uri:       "u1",
		Name:      "a1",
	})
	assert.NoError(t, err)
	art2, err := clientManager.ArtifactStore().CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      int32(apiv2beta1.Artifact_Artifact),
		Uri:       "u2",
		Name:      "a2",
	})
	assert.NoError(t, err)

	// Links
	_, err = s.CreateArtifactTask(ctxWithUser(), &apiv2beta1.CreateArtifactTaskRequest{
		ArtifactTask: &apiv2beta1.ArtifactTask{
			ArtifactId: art1.UUID,
			TaskId:     t1.UUID,
			Type:       apiv2beta1.ArtifactTaskType_INPUT,
		},
	})
	assert.NoError(t, err)
	_, err = s.CreateArtifactTask(ctxWithUser(), &apiv2beta1.CreateArtifactTaskRequest{
		ArtifactTask: &apiv2beta1.ArtifactTask{
			ArtifactId: art2.UUID,
			TaskId:     t1.UUID,
			Type:       apiv2beta1.ArtifactTaskType_OUTPUT,
		},
	})
	assert.NoError(t, err)
	_, err = s.CreateArtifactTask(ctxWithUser(), &apiv2beta1.CreateArtifactTaskRequest{
		ArtifactTask: &apiv2beta1.ArtifactTask{
			ArtifactId: art2.UUID,
			TaskId:     t2.UUID,
			Type:       apiv2beta1.ArtifactTaskType_INPUT,
		},
	})
	assert.NoError(t, err)

	return s, clientManager, t1, t2, art1, art2
}

func TestArtifactServer_ListArtifactTasks_FilterByTaskIds(t *testing.T) {
	s, _, t1, _, _, _ := seedArtifactTasks(t)
	resp, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{TaskIds: []string{t1.UUID}, PageSize: 50})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), resp.GetTotalSize())
	assert.Equal(t, 2, len(resp.GetArtifactTasks()))
	assert.Empty(t, resp.GetNextPageToken())
}

func TestArtifactServer_ListArtifactTasks_FilterByArtifactIds(t *testing.T) {
	s, _, _, _, _, art2 := seedArtifactTasks(t)
	resp, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{ArtifactIds: []string{art2.UUID}, PageSize: 50})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), resp.GetTotalSize())
	assert.Equal(t, 2, len(resp.GetArtifactTasks()))
}

func TestArtifactServer_ListArtifactTasks_FilterByRunIds(t *testing.T) {
	s, _, _, t2, _, art2 := seedArtifactTasks(t)
	resp, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{RunIds: []string{serverRunID2}, PageSize: 50})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), resp.GetTotalSize())
	assert.Equal(t, 1, len(resp.GetArtifactTasks()))
	at := resp.GetArtifactTasks()[0]
	assert.Equal(t, art2.UUID, at.GetArtifactId())
	assert.Equal(t, t2.UUID, at.GetTaskId())
}

func TestArtifactServer_ListArtifactTasks_ErrorWhenNoFilters(t *testing.T) {
	s, _, _, _, _, _ := seedArtifactTasks(t)
	_, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{PageSize: 2})
	assert.Error(t, err)
}

func TestArtifactServer_ListArtifactTasks_Pagination_TaskIds(t *testing.T) {
	s, _, t1, _, _, _ := seedArtifactTasks(t)
	page1, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{TaskIds: []string{t1.UUID}, PageSize: 1})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), page1.GetTotalSize())
	assert.Equal(t, 1, len(page1.GetArtifactTasks()))
	assert.NotEmpty(t, page1.GetNextPageToken())

	page2, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{TaskIds: []string{t1.UUID}, PageToken: page1.GetNextPageToken(), PageSize: 1})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), page2.GetTotalSize())
	assert.Equal(t, 1, len(page2.GetArtifactTasks()))
	assert.Empty(t, page2.GetNextPageToken())

	id1 := page1.GetArtifactTasks()[0].GetId()
	id2 := page2.GetArtifactTasks()[0].GetId()
	assert.NotEqual(t, id1, id2)
}
