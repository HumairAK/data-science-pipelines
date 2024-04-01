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

type GCSProviderConfig struct {
	// required
	Credentials *GCSCredentials `json:"credentials"`
	// optional, ordered, the auth config for the first matching prefix is used
	Overrides []GCSOverride `json:"Overrides"`
}

type GCSOverride struct {
	BucketName  string          `json:"bucketName"`
	KeyPrefix   string          `json:"keyPrefix"`
	Credentials *GCSCredentials `json:"credentials"`
}
type GCSCredentials struct {
	// optional
	FromEnv bool `json:"fromEnv"`
	// if FromEnv is False then SecretRef is required
	SecretRef *GCSSecretRef `json:"secretRef"`
}
type GCSSecretRef struct {
	SecretName string `json:"secretName"`
	TokenKey   string `json:"tokenKey"`
}

func (p GCSProviderConfig) ProvideSessionInfo(bucketName, bucketPrefix string) (objectstore.SessionInfo, error) {
	params := map[string]string{}

	if p.Credentials != nil {
		return objectstore.SessionInfo{}, fmt.Errorf("no default credentials provided in provider config")
	}

	params["fromEnv"] = strconv.FormatBool(p.Credentials.FromEnv)
	if !p.Credentials.FromEnv {
		params["secretName"] = p.Credentials.SecretRef.SecretName
		params["tokenKey"] = p.Credentials.SecretRef.TokenKey
	}

	// Set defaults
	sessionInfo := objectstore.SessionInfo{
		Provider: "gcs",
		Params:   params,
	}

	// If there's a matching override, then override defaults with provided configs
	override := p.getOverrideByPrefix(bucketName, bucketPrefix)
	if override != nil {
		if override.Credentials != nil {
			return objectstore.SessionInfo{}, fmt.Errorf("no override credentials provided in provider config")
		}
		params["fromEnv"] = strconv.FormatBool(override.Credentials.FromEnv)
		if !override.Credentials.FromEnv {
			params["secretName"] = override.Credentials.SecretRef.SecretName
			params["tokenKey"] = p.Credentials.SecretRef.TokenKey
		}
	}
	return sessionInfo, nil
}

// getOverrideByPrefix returns first matching bucketname and prefix in overrides
func (p GCSProviderConfig) getOverrideByPrefix(bucketName, prefix string) *GCSOverride {
	for _, override := range p.Overrides {
		if override.BucketName == bucketName && strings.HasPrefix(prefix, override.KeyPrefix) {
			return &override
		}
	}
	return nil
}
