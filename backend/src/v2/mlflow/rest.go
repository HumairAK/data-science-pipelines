package mlflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/v2/mlflow/types"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"time"
)

func (m *MetadataMLFlow) createRun(runName string, tags []types.RunTag, experimentID string) (*types.Run, error) {
	// Create struct with parameters
	payload := types.CreateRunRequest{
		ExperimentId: experimentID,
		RunName:      runName,
		StartTime:    time.Now().UnixMilli(),
		Tags:         tags,
	}

	// Marshal to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	_, body, err := DoRequest("POST", fmt.Sprintf("%s/runs/create", m.trackingServerHost), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer YOUR_TOKEN_HERE",
	})
	if err != nil {
		return nil, err
	}

	runResponse := &types.CreateRunResponse{}
	err = json.Unmarshal(body, runResponse)
	if err != nil {
		glog.Errorf("Failed to unmarshal: %v", err)
		return nil, err
	}
	return &runResponse.Run, nil
}

func (m *MetadataMLFlow) getRun(runID string) (*types.GetRunResponse, error) {
	// Create struct with parameters
	payload := types.GetRunRequest{
		RunID: runID,
	}

	// Marshal to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	_, body, err := DoRequest("GET", fmt.Sprintf("%s/runs/get", m.trackingServerHost), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer YOUR_TOKEN_HERE",
	})
	if err != nil {
		return nil, err
	}
	runResponse := &types.GetRunResponse{}
	err = json.Unmarshal(body, runResponse)
	if err != nil {
		glog.Errorf("Failed to unmarshal: %v", err)
		return nil, err
	}
	return runResponse, nil
}

func (m *MetadataMLFlow) updateRun(runID string, runName *string, status *types.RunStatus, endTime *int64) (*types.UpdateRunResponse, error) {
	// Create struct with parameters
	payload := types.UpdateRunRequest{
		RunId:   runID,
		RunUUID: runID,
	}

	// Optional
	if runName != nil {
		payload.RunName = *runName
	}
	if status != nil {
		payload.Status = *status
	}
	if endTime != nil {
		payload.EndTime = *endTime
	}

	// Marshal to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	_, body, err := DoRequest("POST", fmt.Sprintf("%s/runs/update", m.trackingServerHost), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer YOUR_TOKEN_HERE",
	})
	if err != nil {
		return nil, err
	}
	typedResp := &types.UpdateRunResponse{}
	err = json.Unmarshal(body, typedResp)
	if err != nil {
		glog.Errorf("Failed to unmarshal: %v", err)
		return nil, err
	}
	return typedResp, nil
}

func (m *MetadataMLFlow) logParam(runID, runUUID, key, value string) error {
	payload := types.LogParamRequest{
		RunId:   runID,
		RunUUID: runUUID,
		Key:     key,
		Value:   value,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, _, err = DoRequest("POST", fmt.Sprintf("%s/runs/log-parameter", m.trackingServerHost), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer YOUR_TOKEN_HERE",
	})
	if err != nil {
		return err
	}
	return nil
}

func (m *MetadataMLFlow) searchExperiments(maxResults int64, pageToken string, filter string, orderBy []string, viewType types.ViewType) ([]types.Experiment, error) {
	return nil, nil
}

func (m *MetadataMLFlow) searchRuns(experimentIds []string, maxResults int64, pageToken string, filter string, orderBy []string, viewType types.ViewType) ([]types.Run, error) {
	payload := types.SearchRunRequest{
		ExperimentIds: experimentIds,
		Filter:        filter,
		RunViewType:   viewType,
		MaxResults:    maxResults,
		PageToken:     pageToken,
		OrderBy:       orderBy,
	}

	// Marshal to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	_, body, err := DoRequest("POST", fmt.Sprintf("%s/runs/search", m.trackingServerHost), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer YOUR_TOKEN_HERE",
	})
	if err != nil {
		return nil, err
	}

	runResponse := &types.SearchRunResponse{}
	err = json.Unmarshal(body, runResponse)
	if err != nil {
		glog.Errorf("Failed to unmarshal: %v", err)
		return nil, err
	}
	return runResponse.Runs, nil
}

func (m *MetadataMLFlow) createExperiment(name string, tags []types.ExperimentTag) (*string, error) {
	// Create struct with parameters
	payload := types.CreateExperimentRequest{
		Name: name,
		Tags: tags,
	}

	// Marshal to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	_, body, err := DoRequest("POST", fmt.Sprintf("%s/experiments/create", m.trackingServerHost), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer YOUR_TOKEN_HERE",
	})
	if err != nil {
		return nil, err
	}

	experimentResponse := &types.CreateExperimentResponse{}
	err = json.Unmarshal(body, experimentResponse)
	if err != nil {
		glog.Errorf("Failed to unmarshal: %v", err)
		return nil, err
	}
	return &experimentResponse.ExperimentId, nil
}

func DoRequest(method, url string, body []byte, headers map[string]string) (*http.Response, []byte, error) {
	glog.Infof("------------------------------------")
	glog.Infof("Sending %s request to %s", method, url)
	glog.Infof("With Payload: %s", string(body))

	client := &http.Client{}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers if provided
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode == 404 {
		return nil, []byte{}, errors.New(string(body))
	}

	glog.Infof("Request successfully sent. With Status received: %s", resp.Status)
	glog.Infof("Response Body: %s", string(body))
	glog.Infof("------------------------------------")

	return resp, respBody, nil
}
