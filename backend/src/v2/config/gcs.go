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
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"strings"
)

type GCSProviderConfig struct {
	Credentials *GCSCredentials `json:"credentials"`
	// optional, ordered, the auth config for the first matching prefix is used
	Overrides []GCSOverrides `json:"Overrides"`
}

type GCSOverrides struct {
	BucketName  string          `json:"bucketName"`
	KeyPrefix   string          `json:"keyPrefix"`
	Credentials *GCSCredentials `json:"credentials"`
}
type GCSCredentials struct {
	// optional
	FromeEnv  bool          `json:"fromEnv"`
	SecretRef *GCSSecretRef `json:"secretRef"`
}
type GCSSecretRef struct {
	SecretName string `json:"secretName"`
	TokenKey   string `json:"tokenKey"`
}

func (p *GCSProviderConfig) ProvideSessionInfo(bucketName, bucketPrefix string) (objectstore.SessionInfo, error) {
	//TODO implement me
	panic("implement me")
}

// getBucketAuthByPrefix returns first matching bucketname and prefix in authConfigs
func (p *GCSProviderConfig) GetBucketAuthByPrefix() (objectstore.SessionInfo, error) {
	for _, authConfig := range authConfigs {
		if authConfig.BucketName == bucketName && strings.HasPrefix(prefix, authConfig.KeyPrefix) {
			return &authConfig
		}
	}
	return nil
}
