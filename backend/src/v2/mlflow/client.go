package mlflow

import (
	"context"
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/mlflow/types"
	"strconv"
)

const (
	MlflowParentRunId = "mlflow.parentRunId" // Parent DAG Execution ID.
)

type MetadataMLFlow struct {
	trackingServerHost string
	experimentID       string
}

func (m *MetadataMLFlow) CreatePipelineRun(ctx context.Context, runName, pipelineName, runID, namespace, runResource, pipelineRoot, storeSessionInfo string) (*metadata.Pipeline, error) {
	tags := []map[string]string{
		{
			"key":   "kfpRunID",
			"value": runID,
		},
		{
			"key":   "keyNamespace",
			"value": namespace,
		},
		{
			"key":   "keyResourceName",
			"value": runResource,
		},
		{
			"key":   "keyPipelineRoot",
			"value": metadata.GenerateOutputURI(pipelineRoot, []string{pipelineName, runID}, true),
		},
		{
			"key":   "keyStoreSessionInfo",
			"value": storeSessionInfo,
		},
	}
	err := m.CreateRun(runName, tags)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (m *MetadataMLFlow) CreateExecution(ctx context.Context, pipeline *metadata.Pipeline, config *metadata.ExecutionConfig, experimentID string) (*metadata.Execution, error) {
	// Create nested execution run
	if config.ParentDagID != 0 {
		return nil, fmt.Errorf("ParentDag Id required")
	}

	KFPParentRunID := strconv.FormatInt(config.ParentDagID, 10)
	filterString := fmt.Sprintf("tags.kfpRunID = '%s'", KFPParentRunID)
	// Get MLFLow run id by searching parentID
	runsResp, err := m.SearchRuns([]string{experimentID}, 10, "", filterString, []string{"DESC"}, "")
	if err != nil {
		return nil, err
	}

	// should be the first (and only) run:
	runs := runsResp.Runs
	if len(runs) == 0 {
		return nil, fmt.Errorf("No runs found for experiment %s", experimentID)
	} else if len(runs) > 1 {
		return nil, fmt.Errorf("Multiple runs found for experiment %s", experimentID)
	}

	run := runs[0]

	var tags []map[string]string
	tags = append(tags, map[string]string{
		MlflowParentRunId: run.Info.RunID,
	})

	createResp, err := m.CreateRun(config.Name, tags)
	if err != nil {
		return nil, err
	}

	createdRun := createResp.Run

	// TODO: log parameters
	if config.InputParameters != nil {
		var result []types.LogParamRequest
		for key, val := range config.InputParameters {
			// Use val.String() to get a printable representation of the structpb.Value
			// Alternatively, use val.GetStringValue() if you're sure it's a string type
			result = append(result, types.LogParamRequest{
				Key:   key,
				Value: fmt.Sprintf("%v", val.AsInterface()),
			})
		}
	}

	return nil, nil
}

func NewMetadataMLFlow(h string, e string) (*MetadataMLFlow, error) {
	return &MetadataMLFlow{
		trackingServerHost: h,
		experimentID:       e,
	}, nil
}
