// Copyright 2024 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"strconv"
	"strings"
)

type S3ProviderConfig struct {
	Endpoint    string         `json:"endpoint"`
	Credentials *S3Credentials `json:"credentials"`
	// optional for any non aws s3 provider
	Region string `json:"region"`
	// optional
	DisableSSL *bool `json:"disableSSL"`
	// optional, ordered, the auth config for the first matching prefix is used
	Overrides []S3Override `json:"Overrides"`
}
type S3Credentials struct {
	// optional
	FromEnv bool `json:"fromEnv"`
	// if FromEnv is False then SecretRef is required
	SecretRef *S3SecretRef `json:"secretRef"`
}
type S3Override struct {
	Endpoint string `json:"endpoint"`
	// optional for any non aws s3 provider
	Region string `json:"region"`
	// optional
	DisableSSL *bool  `json:"disableSSL"`
	BucketName string `json:"bucketName"`
	KeyPrefix  string `json:"keyPrefix"`
	// required
	Credentials *S3Credentials `json:"credentials"`
}
type S3SecretRef struct {
	SecretName   string `json:"secretName"`
	AccessKeyKey string `json:"accessKeyKey"`
	SecretKeyKey string `json:"secretKeyKey"`
}

func (p S3ProviderConfig) ProvideSessionInfo(bucketName, bucketPrefix string) (objectstore.SessionInfo, error) {
	invalidConfigErr := func(err error) error {
		return fmt.Errorf("invalid provider config: %w", err)
	}

	params := map[string]string{}
	params["region"] = p.Region
	params["endpoint"] = p.Endpoint

	if p.DisableSSL == nil {
		params["disableSSL"] = "false"
	} else {
		params["disableSSL"] = strconv.FormatBool(*p.DisableSSL)
	}

	if p.Credentials == nil {
		return objectstore.SessionInfo{}, invalidConfigErr(fmt.Errorf("no default credentials provided in provider config"))
	}

	params["fromEnv"] = strconv.FormatBool(p.Credentials.FromEnv)
	if !p.Credentials.FromEnv {
		params["secretName"] = p.Credentials.SecretRef.SecretName
		params["accessKeyKey"] = p.Credentials.SecretRef.AccessKeyKey
		params["secretKeyKey"] = p.Credentials.SecretRef.SecretKeyKey
	}

	// Set defaults
	sessionInfo := objectstore.SessionInfo{
		Provider: "s3",
		Params:   params,
	}

	// If there's a matching override, then override defaults with provided configs
	override := p.getOverrideByPrefix(bucketName, bucketPrefix)
	if override != nil {
		if override.Endpoint != "" {
			sessionInfo.Params["endpoint"] = override.Endpoint
		}
		if override.Region != "" {
			sessionInfo.Params["region"] = override.Region
		}
		if override.DisableSSL != nil {
			sessionInfo.Params["disableSSL"] = strconv.FormatBool(*override.DisableSSL)
		}
		if override.Credentials == nil {
			return objectstore.SessionInfo{}, fmt.Errorf("no override credentials provided in provider config")
		}
		params["fromEnv"] = strconv.FormatBool(override.Credentials.FromEnv)
		if !override.Credentials.FromEnv {
			params["secretName"] = override.Credentials.SecretRef.SecretName
			params["accessKeyKey"] = override.Credentials.SecretRef.AccessKeyKey
			params["secretKeyKey"] = override.Credentials.SecretRef.SecretKeyKey
		}
	}
	return sessionInfo, nil
}

// getOverrideByPrefix returns first matching bucketname and prefix in overrides
func (p S3ProviderConfig) getOverrideByPrefix(bucketName, prefix string) *S3Override {
	for _, override := range p.Overrides {
		if override.BucketName == bucketName && strings.HasPrefix(prefix, override.KeyPrefix) {
			return &override
		}
	}
	return nil
}
