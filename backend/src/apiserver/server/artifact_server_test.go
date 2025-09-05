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

func TestArtifactServer_CreateGetUpdateListArtifacts(t *testing.T) {
	// Enable multi-user mode so namespace is honored
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Create
	createReq := &apiv2beta1.CreateArtifactRequest{Artifact: &apiv2beta1.Artifact{
		Namespace: "ns1",
		Type:      apiv2beta1.Artifact_Model,
		Uri:       "gs://b/f",
		Name:      "a1",
	}}

	created, err := s.CreateArtifact(ctxWithUser(), createReq)
	assert.NoError(t, err)
	assert.NotEmpty(t, created.GetArtifactId())
	assert.Equal(t, "ns1", created.GetNamespace())
	assert.Equal(t, apiv2beta1.Artifact_Model, created.GetType())

	// Get
	got, err := s.GetArtifact(ctxWithUser(), &apiv2beta1.GetArtifactRequest{ArtifactId: created.GetArtifactId()})
	assert.NoError(t, err)
	assert.Equal(t, created.GetArtifactId(), got.GetArtifactId())

	// Update
	got.Name = "a1-upd"
	got.Type = apiv2beta1.Artifact_Dataset
	upd, err := s.UpdateArtifact(ctxWithUser(), &apiv2beta1.UpdateArtifactRequest{Artifact: got})
	assert.NoError(t, err)
	assert.Equal(t, "a1-upd", upd.GetName())
	assert.Equal(t, apiv2beta1.Artifact_Dataset, upd.GetType())

	// List
	listResp, err := s.ListArtifacts(ctxWithUser(), &apiv2beta1.ListArtifactRequest{Namespace: "ns1", PageSize: 10})
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

func TestArtifactServer_ListArtifactTasks_LogAndGetMetrics(t *testing.T) {
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

	// Create an artifact
	art, err := s.CreateArtifact(
		context.Background(),
		&apiv2beta1.CreateArtifactRequest{
			Artifact: &apiv2beta1.Artifact{
				Namespace: "ns1",
				Type:      apiv2beta1.Artifact_Artifact,
				Uri:       "u",
				Name:      "a",
			},
		})
	assert.NoError(t, err)

	// ListArtifactTasks requires at least one filter
	_, err = s.ListArtifactTasks(context.Background(), &apiv2beta1.ListArtifactTasksRequest{})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())

	// List by artifact id (ResourceManager currently returns empty as not implemented)
	lat, err := s.ListArtifactTasks(context.Background(), &apiv2beta1.ListArtifactTasksRequest{
		ArtifactIds: []string{art.GetArtifactId()},
		PageSize:    10},
	)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), lat.GetTotalSize())
	assert.Equal(t, 0, len(lat.GetArtifactTasks()))

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
	created, err := s.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{Artifact: &apiv2beta1.Artifact{
		Namespace: "ns1",
		Type:      apiv2beta1.Artifact_Artifact,
		Uri:       "u",
		Name:      "a",
	}})
	assert.NoError(t, err)
	assert.Equal(t, "", created.GetNamespace())
}
