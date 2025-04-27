package aimstack

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func (m *MetadataAimstack) CreateRun() error {
	jsonPayload := []byte(`{"name":"John","age":30}`)

	resp, body, err := DoRequest("POST", fmt.Sprintf("%s/runs", m.repo), jsonPayload, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer YOUR_TOKEN_HERE",
	})
	if err != nil {
		panic(err)
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
