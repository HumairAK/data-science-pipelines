package model_registry

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/kubeflow/model-registry/pkg/openapi"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/mlflow/types"
	"io"
	"net/http"
	"strconv"
)

func mrExperimentToModelExperiment(experiment openapi.Experiment) (*model.Experiment, error) {
	nsValue := experiment.GetCustomProperties()["namespace"]
	ns := nsValue.MetadataStringValue.GetStringValue()
	createdAtInSec, err := strconv.ParseInt(experiment.GetCreateTimeSinceEpoch(), 10, 64)
	if err != nil {
		return nil, err
	}
	lastRunCreatedAtInSec, err := strconv.ParseInt(experiment.GetLastUpdateTimeSinceEpoch(), 10, 64)
	if err != nil {
		return nil, err
	}
	experimentModel := &model.Experiment{
		UUID: *experiment.Id,
		Name: experiment.Name,
		// Unix time value (used by MLFlow) is in milliseconds,
		// but the google.protobuf.Timestamp struct expects seconds in its Seconds field.
		CreatedAtInSec: createdAtInSec / 1000,
		// TODO: This is not an exact mapping and can be misleading
		// last update makes more sense and model.experiment should support that
		// lastrun is used to sort kfp ui experiments by last run
		LastRunCreatedAtInSec: lastRunCreatedAtInSec / 1000,
		Namespace:             ns,
	}
	if experiment.Description != nil {
		experimentModel.Description = *experiment.Description
	}

	switch experiment.GetState() {
	case openapi.EXPERIMENTSTATE_LIVE:
		experimentModel.StorageState = model.StorageStateAvailable
	case openapi.EXPERIMENTSTATE_ARCHIVED:
		experimentModel.StorageState = model.StorageStateArchived
	default:
		experimentModel.StorageState = model.StorageStateUnspecified
	}
	return experimentModel, nil
}

// GetExperimentTag return value associated with tag or empty string if tag doesn't exist
func GetExperimentTag(experiment *types.Experiment, tagKey string) string {
	for _, tag := range experiment.Tags {
		if tag.Key == tagKey {
			return tag.Value
		}
	}
	return ""
}

func BuildExperimentNamespaceName(name, namespace string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

// TODO: this is the same as the MLFlow provider code, move this to a general utility
func DoRequest(method, url string, body []byte, headers map[string]string, client *http.Client) (*http.Response, []byte, error) {
	glog.Infof("------------------------------------")
	glog.Infof("Sending %s request to %s", method, url)
	glog.Infof("With Payload: %s", string(body))

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
