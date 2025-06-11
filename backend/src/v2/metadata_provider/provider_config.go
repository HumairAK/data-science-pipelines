package metadata_provider

import (
	"encoding/json"
	"fmt"
	k8score "k8s.io/api/core/v1"
	"os"
)

// MetadataProvider defines the type of metadata provider
type MetadataProvider string

const (
	MetadataProviderMLFlow MetadataProvider = "mlflow"
)

// TODO: This should be the Proto value once defined in experiment.proto for createrequest
type ProviderRuntimeConfig map[string]interface{}

type MetadataProviderConfig struct {
	MLFlow *MLFlow `json:"mlflow"`
}

type MLFlow struct {
	MLFlowServerConfig *MLFlowServerConfig
}
type ExecutionConfig struct {
	ExperimentID string           `json:"experiment_id"`
	EnvVars      []k8score.EnvVar `json:"env_vars"`
}

type MLFlowServerConfig struct {
	Host        string `json:"host"`
	Port        string `json:"port"`
	TLSEnabled  bool   `json:"TLSEnabled"`
	APIPath     string `json:"apiPath"`
	MetricsPath string `json:"metricsPath"`
	// TODO: Add tls cert handling
}

func NewProviderConfig(provider MetadataProvider) (*MetadataProviderConfig, error) {
	switch provider {
	case MetadataProviderMLFlow:
		hostEnv := os.Getenv("MLFLOW_HOST")
		portEnv := os.Getenv("MLFLOW_PORT")
		tlsEnabled := os.Getenv("MLFLOW_TLS_ENABLED")
		if hostEnv == "" {
			return nil, fmt.Errorf("Missing environment variable MLFLOW_HOST")
		}

		var protocol string
		if tlsEnabled == "true" {
			protocol = "https"
		} else {
			protocol = "http"
		}
		var basePath string
		if portEnv != "" {
			basePath = fmt.Sprintf("%s://%s:%s", protocol, hostEnv, portEnv)
		} else {
			basePath = fmt.Sprintf("%s://%s", protocol, hostEnv)
		}
		apiPath := fmt.Sprintf("%s/api/2.0/mlflow", basePath)
		metricsPath := fmt.Sprintf("%s/#/metric", basePath)
		return &MetadataProviderConfig{
			MLFlow: &MLFlow{
				MLFlowServerConfig: &MLFlowServerConfig{
					Host:        hostEnv,
					Port:        portEnv,
					TLSEnabled:  tlsEnabled == "true",
					APIPath:     apiPath,
					MetricsPath: metricsPath,
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("Unknown metadata provider %s", provider)
	}
}

func (p *MetadataProviderConfig) GetProviderConfigJSON() (string, error) {
	jsonBytes, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("failed to marshal provider config: %v", err)
	}
	return string(jsonBytes), nil
}
