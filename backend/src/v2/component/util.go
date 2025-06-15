package component

import (
	"context"
	"fmt"
	v2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
	"io"
	"os"
)

// CopyThisBinary copies the running binary into destination path.
func CopyThisBinary(destination string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("copy this binary to %s: %w", destination, err)
		}
	}()

	path, err := findThisBinary()
	if err != nil {
		return err
	}
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(destination, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o555) // 0o555 -> readable and executable by all
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err = io.Copy(dst, src); err != nil {
		return err
	}
	return dst.Close()
}

func findThisBinary() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("findThisBinary failed: %w", err)
	}
	return path, nil
}

func CreateRunMetadata(
	ctx context.Context,
	providerRunName string,
	cm *client_manager.ClientManager,
	experimentID string,
	runID string,
	ecfg *metadata.ExecutionConfig,
	parentID string,
) (ProviderRunId string, err error) {
	rsc := cm.RunServiceClient()
	kfpRun, err := rsc.GetRun(
		ctx,
		&v2beta1.GetRunRequest{
			RunId: runID,
		})
	if err != nil {
		return "", err
	}
	run, err := cm.MetadataRunProvider().CreateRun(
		experimentID,
		kfpRun,
		providerRunName,
		metadata_provider.PBParamsToRunParameters(ecfg.InputParameters),
		parentID,
	)
	if err != nil {
		return "", err
	}
	ecfg.ProviderRunID = &run.ID
	return run.ID, nil
}
