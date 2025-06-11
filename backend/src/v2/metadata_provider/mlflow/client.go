package mlflow

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/mlflow/types"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"time"
)

type Client struct {
	apiPath     string
	baseHost    string
	metricsPath string
	token       string
}

func NewClient(config *MLFlowServerConfig) (*Client, error) {
	host := config.Host
	port := config.Port
	tlsEnabled := config.TLSEnabled
	var protocol string
	if tlsEnabled == "true" {
		protocol = "https"
	} else {
		protocol = "http"
	}
	var basePath string
	if port != "" {
		basePath = fmt.Sprintf("%s://%s:%s", protocol, host, port)
	} else {
		basePath = fmt.Sprintf("%s://%s", protocol, host)
	}
	apiPath := fmt.Sprintf("%s/api/2.0/mlflow", basePath)
	metricsPath := fmt.Sprintf("%s/#/metric", basePath)

	authToken := os.Getenv("MLFLOW_TRACKING_SERVER_TOKEN")

	return &Client{
		apiPath:     apiPath,
		baseHost:    basePath,
		metricsPath: metricsPath,
		token:       authToken,
	}, nil
}

func (m *Client) createRun(runName string, tags []types.RunTag, experimentID string) (*types.Run, error) {
	payload := types.CreateRunRequest{
		ExperimentId: experimentID,
		RunName:      runName,
		StartTime:    time.Now().UnixMilli(),
		Tags:         tags,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	_, body, err := DoRequest("POST", fmt.Sprintf("%s/runs/create", m.apiPath), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": m.token,
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

func (m *Client) getRun(runID string) (*types.Run, error) {
	payload := types.GetRunRequest{
		RunID: runID,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	_, body, err := DoRequest("GET", fmt.Sprintf("%s/runs/get", m.apiPath), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": m.token,
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
	return &runResponse.Run, nil
}

func (m *Client) updateRun(runID string, runName *string, status *types.RunStatus, endTime *int64) error {
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

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, body, err := DoRequest("POST", fmt.Sprintf("%s/runs/update", m.apiPath), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": m.token,
	})
	if err != nil {
		return err
	}
	typedResp := &types.UpdateRunResponse{}
	err = json.Unmarshal(body, typedResp)
	if err != nil {
		glog.Errorf("Failed to unmarshal: %v", err)
		return err
	}
	return nil
}

func (m *Client) logParam(runID *string, key, value string) error {
	payload := types.LogParamRequest{
		RunId: *runID,
		Key:   key,
		Value: value,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, _, err = DoRequest("POST", fmt.Sprintf("%s/runs/log-parameter", m.apiPath), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": m.token,
	})
	if err != nil {
		return err
	}
	return nil
}

func (m *Client) logMetric(runID, runUUID, key string, value float64) error {
	payload := types.LogMetricRequest{
		RunId:     runID,
		RunUUID:   runUUID,
		Key:       key,
		Value:     value,
		Timestamp: time.Now().UnixMilli(),
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, _, err = DoRequest("POST", fmt.Sprintf("%s/runs/log-metric", m.apiPath), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": m.token,
	})
	if err != nil {
		return err
	}
	return nil
}

func (m *Client) getExperiment(id string) (*types.Experiment, error) {
	payload := types.GetExperimentRequest{
		ExperimentId: id,
	}
	// Marshal to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	_, body, err := DoRequest("GET", fmt.Sprintf("%s/experiments/get", m.apiPath), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": m.token,
	})
	if err != nil {
		return nil, err
	}

	experimentResponse := &types.GetExperimentResponse{}
	err = json.Unmarshal(body, experimentResponse)
	if err != nil {
		glog.Errorf("Failed to unmarshal: %v", err)
		return nil, err
	}
	return &experimentResponse.Experiment, nil
}

func (m *Client) getExperimentByName(name string) (*types.Experiment, error) {
	payload := types.GetExperimentByNameRequest{
		ExperimentName: name,
	}
	// Marshal to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	_, body, err := DoRequest("GET", fmt.Sprintf("%s/experiments/get-by-name", m.apiPath), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": m.token,
	})
	if err != nil {
		return nil, err
	}

	experimentResponse := &types.GetExperimentByNameResponse{}
	err = json.Unmarshal(body, experimentResponse)
	if err != nil {
		glog.Errorf("Failed to unmarshal: %v", err)
		return nil, err
	}
	return &experimentResponse.Experiment, nil
}

func (m *Client) searchExperiments(maxResults int64, pageToken string, filter string, orderBy []string, viewType types.ViewType) ([]types.Experiment, string, error) {

	payload := types.SearchExperimentRequest{
		Filter:     filter,
		ViewType:   viewType,
		MaxResults: maxResults,
		PageToken:  pageToken,
		OrderBy:    orderBy,
	}

	// Marshal to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}

	_, body, err := DoRequest("POST", fmt.Sprintf("%s/experiments/search", m.apiPath), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": m.token,
	})
	if err != nil {
		return nil, "", err
	}

	experimentResponse := &types.SearchExperimentResponse{}
	err = json.Unmarshal(body, experimentResponse)
	if err != nil {
		glog.Errorf("Failed to unmarshal: %v", err)
		return nil, "", err
	}
	return experimentResponse.Experiments, experimentResponse.NextPageToken, nil
}

func (m *Client) searchRuns(
	experimentIds []string,
	maxResults int64,
	pageToken string,
	filter string,
	orderBy []string,
	viewType types.ViewType) ([]types.Run, error) {
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

	_, body, err := DoRequest("POST", fmt.Sprintf("%s/runs/search", m.apiPath), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": m.token,
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

// createExperiment creates experiment
// artifactLocation is optional and is ignored if it's an empty string
// namespace is optional
// returns experiment id
func (m *Client) createExperiment(name, artifactLocation string, tags []types.ExperimentTag) (string, error) {

	payload := types.CreateExperimentRequest{
		Name: name,
		Tags: tags,
	}

	if artifactLocation != "" {
		payload.ArtifactLocation = artifactLocation
	}

	// Marshal to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	_, body, err := DoRequest("POST", fmt.Sprintf("%s/experiments/create", m.apiPath), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": m.token,
	})
	if err != nil {
		return "", err
	}

	experimentResponse := &types.CreateExperimentResponse{}
	err = json.Unmarshal(body, experimentResponse)
	if err != nil {
		glog.Errorf("Failed to unmarshal: %v", err)
		return "", err
	}
	return experimentResponse.ExperimentId, nil
}

func (m *Client) deleteExperiment(id string) error {
	payload := types.DeleteExperimentRequest{
		ExperimentId: id,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, _, err = DoRequest("POST", fmt.Sprintf("%s/experiments/delete", m.apiPath), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": m.token,
	})
	if err != nil {
		return err
	}

	return nil
}

func DoRequest(method, url string, body []byte, headers map[string]string) (*http.Response, []byte, error) {
	glog.Infof("------------------------------------")
	glog.Infof("Sending %s request to %s", method, url)
	glog.Infof("With Payload: %s", string(body))

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		glog.Errorf("Request failed with status code %d: %s", resp.StatusCode, string(respBody))
		return nil, []byte{}, errors.New(string(respBody))
	}

	glog.Infof("Request successfully sent. With Status received: %s", resp.Status)
	glog.Infof("Response Body: %s", string(body))
	glog.Infof("------------------------------------")

	return resp, respBody, nil
}
