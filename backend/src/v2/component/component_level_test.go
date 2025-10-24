package component

import (
	"os"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_LauncherV2_basic(t *testing.T) {
	testDataBytes, err := os.ReadFile("testdata/executorinput.json")
	require.NoError(t, err)
	executorJson := string(testDataBytes)
	mockAPI := kfpapi.NewMockAPI()
	fakeKubernetesClientSet := fake.NewClientset()

	clientManager := client_manager.NewFakeClientManager(fakeKubernetesClientSet, mockAPI)
	opts := &LauncherV2Options{
		Namespace:    "default",
		PodName:      "pod-name",
		PodUID:       "pod-uid",
		PipelineName: "pipeline-name",
	}
	launcher, err := NewLauncherV2(executorJson, []string{"sh", "-c", "echo \"hello world\""}, opts, clientManager)
	require.NoError(t, err)
	require.NotNil(t, launcher)

	// Launcher will take in executor input
	// Launcher will download files
	// run execute
	// and upload files

}
