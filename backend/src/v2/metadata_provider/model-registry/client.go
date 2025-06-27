package mlflow

import (
	"context"
	"fmt"
	"github.com/kubeflow/model-registry/pkg/openapi"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/config"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/mlflow/types"
	"os"
	"strconv"
	"time"
)

type Client struct {
	*openapi.ModelRegistryServiceAPIService
}

// NewClient returns a new MLFlow client.
// Assumes Env vars:
// MLFLOW_TRACKING_SERVER_TOKEN
func NewClient(config config.GenericProviderConfig) (*Client, error) {
	registryConfig, err := ConvertToModelRegistryConfig(config)
	if err != nil {
		return nil, err
	}
	host := registryConfig.Host
	tlsEnabled := registryConfig.TLSEnabled
	var protocol string
	if tlsEnabled == "true" {
		protocol = "https"
	} else {
		protocol = "http"
	}
	authToken := os.Getenv("MODEL_REGISTRY_TOKEN")

	apiConfig := openapi.NewConfiguration()
	apiConfig.Host = host
	apiConfig.DefaultHeader["Authorization"] = authToken
	apiConfig.Scheme = protocol
	apiConfig.Debug = registryConfig.Debug
	apiclient := openapi.NewAPIClient(apiConfig)
	return &Client{
		apiclient.ModelRegistryServiceAPI,
	}, nil
}

func (m *Client) createRun(
	runName string,
	tags *map[string]openapi.MetadataValue,
	description,
	experimentID string) (*openapi.ExperimentRun, error) {
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

	run, resp, err := m.ModelRegistryServiceAPIService.CreateExperimentRun(ctx).ExperimentRunCreate(payload).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to create run: %w", err)
	}
	defer resp.Body.Close()

	return run, nil
}

func (m *Client) getRun(runID string) (*openapi.ExperimentRun, error) {
	return nil, nil
}

func (m *Client) updateRun(runID string, runName *string, status *openapi.ExperimentRunStatus, endTime *int64) error {

	return nil
}

func (m *Client) logParam(runID string, key, value string) error {

	return nil
}

func (m *Client) logMetric(runID, runUUID, key string, value float64) error {

	return nil
}

func (m *Client) logBatch(runID string, metrics []types.Metric, params []types.Param, tags []types.RunTag) error {

	return nil
}

func (m *Client) getExperiment(id string) (*openapi.Experiment, error) {
	return nil, nil
}

func (m *Client) getExperimentByName(name string) (*openapi.Experiment, error) {

	return nil, nil
}

// If status is nil, it defaults to LIVE
func (m *Client) getExperimentByNameNamespace(name, namespace string, status *openapi.ExperimentState) (*openapi.Experiment, error) {
	return nil, nil
}

func (m *Client) createExperiment(name, artifactLocation string, tags []types.ExperimentTag) (*string, error) {
	return nil, nil
}

func (m *Client) deleteExperiment(id string) error {

	return nil
}

func (m *Client) restoreExperiment(id string) error {

	return nil
}

func (m *Client) IsHealthy() error {
	return nil
}
