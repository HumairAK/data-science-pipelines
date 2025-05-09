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

func (m *MetadataMLFlow) CreateRun(runName string, tags []types.RunTag) (*types.Run, error) {
	experimentID := m.experimentID

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

	resp, body, err := DoRequest("POST", fmt.Sprintf("%s/runs/create", m.trackingServerHost), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer YOUR_TOKEN_HERE",
	})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 404 {
		return nil, errors.New(string(body))
	}
	fmt.Println("Status:", resp.Status)
	fmt.Println("Body:", string(body))

	runResponse := &types.CreateRunResponse{}
	err = json.Unmarshal(body, runResponse)
	if err != nil {
		glog.Errorf("Failed to unmarshal: %v", err)
		return nil, err
	}
	return &runResponse.Run, nil
}

func (m *MetadataMLFlow) GetRun(runID string) (*types.GetRunResponse, error) {
	// Create struct with parameters
	payload := types.GetRunRequest{
		RunID: runID,
	}

	// Marshal to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, body, err := DoRequest("GET", fmt.Sprintf("%s/runs/get", m.trackingServerHost), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer YOUR_TOKEN_HERE",
	})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 404 {
		return nil, errors.New(string(body))
	}
	var runResponse types.GetRunResponse
	err = json.Unmarshal(body, &runResponse)
	if err != nil {
		glog.Errorf("Failed to unmarshal: %v", err)
	}
	return &runResponse, nil
}

func (m *MetadataMLFlow) UpdateRun(runID string, runName *string, status *types.RunStatus, endTime *int64) (*types.UpdateRunResponse, error) {
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

	resp, body, err := DoRequest("POST", fmt.Sprintf("%s/runs/update", m.trackingServerHost), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer YOUR_TOKEN_HERE",
	})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 404 {
		return nil, errors.New(string(body))
	}
	var typedResp *types.UpdateRunResponse
	err = json.Unmarshal(body, typedResp)
	if err != nil {
		glog.Errorf("Failed to unmarshal: %v", err)
	}
	return typedResp, nil
}

func (m *MetadataMLFlow) LogParam(runID, runUUID, key, value string) error {
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
	resp, body, err := DoRequest("POST", fmt.Sprintf("%s/runs/log-parameter", m.trackingServerHost), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer YOUR_TOKEN_HERE",
	})
	if err != nil {
		return err
	}
	if resp.StatusCode == 404 {
		return errors.New(string(body))
	}
	return nil
}

func (m *MetadataMLFlow) SearchExperiments(maxResults int64, pageToken string, filter string, orderBy []string, viewType types.ViewType) (*types.SearchRunResponse, error) {
	return nil, nil
}

func (m *MetadataMLFlow) SearchRuns(experimentIds []string, maxResults int64, pageToken string, filter string, orderBy []string, viewType types.ViewType) ([]types.Run, error) {
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

	resp, body, err := DoRequest("POST", fmt.Sprintf("%s/runs/search", m.trackingServerHost), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer YOUR_TOKEN_HERE",
	})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 404 {
		return nil, errors.New(string(body))
	}

	var runResponse types.SearchRunResponse
	err = json.Unmarshal(body, &runResponse)
	if err != nil {
		glog.Errorf("Failed to unmarshal: %v", err)
	}
	return runResponse.Runs, nil
}

func DoRequest(method, url string, body []byte, headers map[string]string) (*http.Response, []byte, error) {
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

	return resp, respBody, nil
}
