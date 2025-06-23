package config

import (
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
)

// MetadataProvider defines the type of metadata provider
type MetadataProvider string

type ProviderConfig struct {
	MetadataProviderName MetadataProvider      `json:"MetadataProviderName"`
	SupportNestedRuns    string                `json:"SupportNestedRuns"`
	AdditionalConfig     GenericProviderConfig `json:"AdditionalConfig"`
}

type GenericProviderConfig common.UnstructuredJSON
