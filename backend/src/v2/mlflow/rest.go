package mlflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"time"
)

type RunPayload struct {
	ExperimentID string `json:"experiment_id"`
	RunName      string `json:"run_name"`
	StartTime    string `json:"start_time"`
}

func (m *MetadataMLFlow) CreateRun(runName string) error {
	// Parameter values
	experimentID := m.experimentID

	// Create struct with parameters
	payload := RunPayload{
		ExperimentID: experimentID,
		RunName:      runName,
		StartTime:    fmt.Sprintf("%d", time.Now().UnixMilli()),
	}

	// Marshal to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, body, err := DoRequest("POST", fmt.Sprintf("%s/runs/create", m.trackingServerHost), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer YOUR_TOKEN_HERE",
	})
	if err != nil {
		return err
	}
	if resp.StatusCode == 404 {
		return errors.New(string(body))
	}
	fmt.Println("Status:", resp.Status)
	fmt.Println("Body:", string(body))
	return nil
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
