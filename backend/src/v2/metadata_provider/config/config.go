package config

import (
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	k8score "k8s.io/api/core/v1"
)

// MetadataProvider defines the type of metadata provider
type MetadataProvider string

type ProviderConfig struct {
	MetadataProviderName MetadataProvider      `json:"MetadataProviderName"`
	EnvironmentVariables []k8score.EnvVar      `json:"EnvironmentVariables"`
	AdditionalConfig     GenericProviderConfig `json:"AdditionalConfig"`
}

type GenericProviderConfig common.UnstructuredJSON
