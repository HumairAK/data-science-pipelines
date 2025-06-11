package mlflow

import (
	k8score "k8s.io/api/core/v1"
)

type MLFlowConfig struct {
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
