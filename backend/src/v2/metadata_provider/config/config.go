package config

import (
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	k8score "k8s.io/api/core/v1"
)

// MetadataProvider defines the type of metadata provider
type MetadataProvider string

type ProviderConfig struct {
	MetadataProviderName MetadataProvider      `json:"MetadataProviderName"`
	SupportNestedRuns    string                `json:"SupportNestedRuns"`
	AdditionalConfig     GenericProviderConfig `json:"AdditionalConfig"`
	PipelineEnv          []k8score.EnvVar      `json:"PipelineEnv"`
}

type GenericProviderConfig common.UnstructuredJSON
