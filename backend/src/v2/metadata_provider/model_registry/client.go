package model_registry

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/kubeflow/model-registry/pkg/openapi"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/config"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/mlflow/types"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Client struct {
	APIService                 *openapi.ModelRegistryServiceAPIService
	APIConfig                  *openapi.Configuration
	DefaultArtifactTrackingURI string
	Host                       string
}

// NewClient returns a new MR client.
func NewClient(config config.GenericProviderConfig) (*Client, error) {
	registryConfig, err := ConvertToModelRegistryConfig(config)
	if err != nil {
		return nil, err
	}
	host := registryConfig.Host
	tlsEnabled := registryConfig.TLSEnabled
	var protocol string
	if tlsEnabled {
		protocol = "https"
	} else {
		protocol = "http"
	}
	authToken := os.Getenv(registryConfig.TokenEnvVarName)
	if authToken == "" {
		return nil, errors.New(fmt.Sprintf("Model Registry config env var: %s is not set", registryConfig.TokenEnvVarName))
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: registryConfig.InsecureSkipVerify},
	}
	httpClient := &http.Client{Transport: tr}

	apiConfig := openapi.NewConfiguration()
	apiConfig.Host = host
	apiConfig.DefaultHeader["Authorization"] = fmt.Sprintf("Bearer %s", authToken)
	apiConfig.Scheme = protocol
	apiConfig.Debug = registryConfig.Debug
	apiConfig.HTTPClient = httpClient
	apiclient := openapi.NewAPIClient(apiConfig)

	return &Client{
		APIService:                 apiclient.ModelRegistryServiceAPI,
		Host:                       host,
		APIConfig:                  apiConfig,
		DefaultArtifactTrackingURI: registryConfig.DefaultArtifactURI,
	}, nil
}

func (m *Client) createRun(
	runName string,
	tags *map[string]openapi.MetadataValue,
	description,
	experimentID, parentRunID string) (*openapi.ExperimentRun, error) {
	ctx := context.Background()

	nowDate := strconv.FormatInt(time.Now().UnixMilli(), 10)
	payload := openapi.ExperimentRunCreate{
		CustomProperties:    tags,
		Description:         &description,
		ExternalId:          nil,
		Name:                &runName,
		EndTimeSinceEpoch:   nil,
		Status:              nil,
		State:               nil,
		Owner:               nil,
		ExperimentId:        experimentID,
		StartTimeSinceEpoch: &nowDate,
	}

	if parentRunID != "" {
		payload.Owner = &parentRunID
	}

	run, resp, err := m.APIService.CreateExperimentRun(ctx).ExperimentRunCreate(payload).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to create run: %w", err)
	}
	defer resp.Body.Close()

	return run, nil
}

func (m *Client) getRun(runID string) (*openapi.ExperimentRun, error) {
	ctx := context.Background()

	run, resp, err := m.APIService.GetExperimentRun(ctx, runID).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to update run: %w", err)
	}
	defer resp.Body.Close()

	err = checkExecuteStatus(resp)
	if err != nil {
		return nil, err
	}

	return run, nil
}

func (m *Client) updateRun(
	runID string,
	status *openapi.ExperimentRunStatus,
	endTime *int64,
	params *map[string]openapi.MetadataValue,
	tags *map[string]openapi.MetadataValue,
) error {
	ctx := context.Background()
	if status == nil && endTime == nil && params == nil && tags == nil {
		return fmt.Errorf("no update fields provided")
	}
	payload := openapi.ExperimentRunUpdate{}
	if status != nil {
		payload.Status = status
	}
	if endTime != nil {
		endTimeConverted := strconv.FormatInt(*endTime, 10)
		payload.EndTimeSinceEpoch = &endTimeConverted
	}
	if params != nil {
		payload.CustomProperties = params
	}
	if tags != nil {
		payload.CustomProperties = tags
	}
	_, resp, err := m.APIService.UpdateExperimentRun(ctx, runID).ExperimentRunUpdate(payload).Execute()
	if err != nil {
		return fmt.Errorf("failed to update run: %w", err)
	}
	defer resp.Body.Close()

	err = checkExecuteStatus(resp)
	if err != nil {
		return err
	}

	return nil
}

func (m *Client) logParam(runID, key, value string) (*openapi.Artifact, error) {
	artifactType := "parameter"
	parameterType := openapi.PARAMETERTYPE_STRING
	return m.logArtifact(runID, openapi.ArtifactCreate{
		ParameterCreate: &openapi.ParameterCreate{
			ArtifactType:  &artifactType,
			Name:          &key,
			Value:         &value,
			ParameterType: &parameterType,
		},
	})
}

func (m *Client) logMetric(runID, key string, value float64, step int64) (*openapi.Artifact, error) {
	artifactType := "metric"
	payload := openapi.ArtifactCreate{
		MetricCreate: &openapi.MetricCreate{
			ArtifactType: &artifactType,
			Name:         &key,
			Value:        &value,
			Step:         &step,
		},
	}
	return m.logArtifact(runID, payload)
}

func (m *Client) logArtifact(runID string, payload openapi.ArtifactCreate) (*openapi.Artifact, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	_, body, err := DoRequest(
		"POST",
		fmt.Sprintf("%s/experiment_runs/%s/artifacts", m.Host, runID),
		jsonPayload,
		map[string]string{
			"Content-Type": "application/json",
		},
		m.APIConfig.HTTPClient,
	)
	if err != nil {
		return nil, err
	}
	artifact := &openapi.Artifact{}
	err = json.Unmarshal(body, artifact)
	if err != nil {
		glog.Errorf("Failed to unmarshal: %v", err)
		return nil, err
	}
	return artifact, nil
}

func (m *Client) logBatch(
	runID string,
	metrics []types.Metric,
	params []types.Param,
	tags *map[string]openapi.MetadataValue,
) error {
	// Todo: Update to batching when MR supports it

	if tags != nil && len(*tags) > 0 {
		err := m.updateRun(runID, nil, nil, nil, tags)
		if err != nil {
			return err
		}
	}

	for _, metric := range metrics {
		_, err := m.logMetric(runID, metric.Key, metric.Value, metric.Step)
		if err != nil {
			return err
		}
	}
	for _, param := range params {
		_, err := m.logParam(runID, param.Key, param.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

func checkExecuteStatus(resp *http.Response) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		bodyString := string(body)
		glog.Errorf("Request failed with status code %d: %s", resp.StatusCode, bodyString)
		return errors.New(bodyString)
	}
	return nil
}

func (m *Client) getExperiment(id string) (*openapi.Experiment, error) {
	ctx := context.Background()

	experiment, resp, err := m.APIService.GetExperiment(ctx, id).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to check health: %w", err)
	}
	defer resp.Body.Close()
	err = checkExecuteStatus(resp)
	if err != nil {
		return nil, err
	}
	return experiment, nil
}

func (m *Client) getExperimentByName(name string) (*openapi.Experiment, error) {
	ctx := context.Background()
	experiment, resp, err := m.APIService.FindExperiment(ctx).Name(name).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to check health: %w", err)
	}
	defer resp.Body.Close()
	err = checkExecuteStatus(resp)
	if err != nil {
		return nil, err
	}
	return experiment, nil
}

// If status is nil, it lists all experiments
func (m *Client) getExperiments(
	pageSize int64,
	nextPageToken string,
	orderBy string,
	sortOrder string,
	status *openapi.ExperimentState,
) ([]openapi.Experiment, string, error) {
	ctx := context.Background()
	pageSizeStr := strconv.FormatInt(pageSize, 10)

	experimentsList, resp, err := m.APIService.
		GetExperiments(ctx).
		PageSize(pageSizeStr).
		NextPageToken(nextPageToken).
		SortOrder("ASC"). // TODO: support sort and orderby
		Execute()
	if err != nil {
		return nil, "", fmt.Errorf("failed to check health: %w", err)
	}
	defer resp.Body.Close()
	err = checkExecuteStatus(resp)
	if err != nil {
		return nil, "", err
	}

	// TODO: filtering is not supported in the API yet. So we just filter on the api server side
	// this is not great because some pages might not be equal in length and ship the pageSize
	// for poc it's enough for now
	if status != nil && *status != "" {
		var experimentsListFiltered []openapi.Experiment
		for _, experiment := range experimentsList.Items {
			if experiment.State != nil && *experiment.State == *status {
				experimentsListFiltered = append(experimentsListFiltered, experiment)
			}
		}
		return experimentsListFiltered, experimentsList.NextPageToken, nil
	}

	return experimentsList.Items, experimentsList.NextPageToken, nil
}

func (m *Client) createExperiment(name, description string, tags *map[string]openapi.MetadataValue) (*openapi.Experiment, error) {
	ctx := context.Background()
	payload := openapi.ExperimentCreate{
		Name:             name,
		Description:      &description,
		CustomProperties: tags,
	}
	experiment, resp, err := m.APIService.CreateExperiment(ctx).ExperimentCreate(payload).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to check health: %w", err)
	}
	defer resp.Body.Close()
	err = checkExecuteStatus(resp)
	if err != nil {
		return nil, err
	}
	return experiment, nil
}

func (m *Client) deleteExperiment(id string) error {
	ctx := context.Background()
	newState := openapi.EXPERIMENTSTATE_ARCHIVED
	_, resp, err := m.APIService.UpdateExperiment(ctx, id).ExperimentUpdate(openapi.ExperimentUpdate{
		State: &newState,
	}).Execute()
	if err != nil {
		return fmt.Errorf("failed to delete experiment: %w", err)
	}
	defer resp.Body.Close()
	err = checkExecuteStatus(resp)
	if err != nil {
		return err
	}
	return nil
}

func (m *Client) restoreExperiment(id string) error {
	ctx := context.Background()
	newState := openapi.EXPERIMENTSTATE_LIVE
	_, resp, err := m.APIService.UpdateExperiment(ctx, id).ExperimentUpdate(openapi.ExperimentUpdate{
		State: &newState,
	}).Execute()
	if err != nil {
		return fmt.Errorf("failed to check health: %w", err)
	}
	defer resp.Body.Close()
	err = checkExecuteStatus(resp)
	if err != nil {
		return err
	}
	return nil
}

func (m *Client) IsHealthy() error {
	ctx := context.Background()
	_, resp, err := m.APIService.GetExperiments(ctx).Execute()
	if err != nil {
		return fmt.Errorf("failed to check health: %w", err)
	}
	defer resp.Body.Close()
	err = checkExecuteStatus(resp)
	if err != nil {
		return err
	}
	return nil
}
