package component

import (
	"os"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/stretchr/testify/require"
)

func Test_LauncherV2_basic(t *testing.T) {
	testDataBytes, err := os.ReadFile("testdata/executorinput.json")
	require.NoError(t, err)
	executorJson := string(testDataBytes)
	mockDriverAPI := NewMockDriverAPI()

	clientManager := client_manager.NewFakeClientManager(nil, nil, nil)
	launcheer, err := NewLauncherV2(executorJson, []string{"sh", "-c", "echo \"hello world\""}, nil, nil)
	require.NoError(t, err)
	// Launcher will take in executor input
	// Launcher will download files
	// run execute
	// and upload files

}
