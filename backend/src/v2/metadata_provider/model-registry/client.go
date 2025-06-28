package mlflow

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/kubeflow/model-registry/pkg/openapi"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/config"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Client struct {
	APIService *openapi.ModelRegistryServiceAPIService
	Host       string
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
		APIService: apiclient.ModelRegistryServiceAPI,
		Host:       host,
	}, nil
}

func (m *Client) createRun(
	runName string,
	tags *map[string]openapi.MetadataValue,
	description,
	experimentID,
	parentID string) (*openapi.ExperimentRun, error) {
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

	run, resp, err := m.APIService.CreateExperimentRun(ctx).ExperimentRunCreate(payload).Execute()
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

func (m *Client) logBatch(
	runID string,
	metrics []openapi.Metric,
	params *map[string]openapi.MetadataValue,
	tags *map[string]openapi.MetadataValue,
) error {

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
	experiment, resp, err := m.APIService.FindExperiment(ctx).ExternalId(id).Execute()
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
	if status != nil {
		var experimentsListFiltered []openapi.Experiment
		for _, experiment := range experimentsList.Items {
			if experiment.State != nil && experiment.State == status {
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
