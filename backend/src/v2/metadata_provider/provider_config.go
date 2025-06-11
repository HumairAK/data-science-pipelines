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
	Host       string `json:"host"`
	Port       string `json:"port"`
	TLSEnabled string `json:"TLSEnabled"`
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

		return &MetadataProviderConfig{
			MLFlow: &MLFlow{
				MLFlowServerConfig: &MLFlowServerConfig{
					Host:       hostEnv,
					Port:       portEnv,
					TLSEnabled: tlsEnabled,
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
