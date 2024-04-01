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
	FromEnv   bool         `json:"fromEnv"`
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
	// Get defaults
	sessionInfo := objectstore.SessionInfo{
		Provider: "s3",
		Region:   p.Region,
		Endpoint: p.Endpoint,
		S3CredentialsSecret: objectstore.S3CredentialsSecret{
			FromEnv:      p.Credentials.FromEnv,
			SecretName:   p.Credentials.SecretRef.SecretName,
			AccessKeyKey: p.Credentials.SecretRef.AccessKeyKey,
			SecretKeyKey: p.Credentials.SecretRef.SecretKeyKey,
		},
	}
	if p.DisableSSL == nil {
		sessionInfo.DisableSSL = false
	} else {
		sessionInfo.DisableSSL = *p.DisableSSL
	}
	// If there'p a matching override, then override defaults with provided configs
	override := p.getBucketAuthByPrefix(bucketName, bucketPrefix)
	if override != nil {
		if override.Endpoint != "" {
			sessionInfo.Endpoint = override.Endpoint
		}
		if override.Region != "" {
			sessionInfo.Region = override.Region
		}
		if override.DisableSSL != nil {
			sessionInfo.DisableSSL = *p.DisableSSL
		}
		if override.Credentials != nil {
			return objectstore.SessionInfo{}, fmt.Errorf("no override credentials provided in provider config")
		}
		sessionInfo.S3CredentialsSecret = objectstore.S3CredentialsSecret{
			FromEnv:      override.Credentials.FromEnv,
			SecretName:   override.Credentials.SecretRef.SecretName,
			AccessKeyKey: override.Credentials.SecretRef.AccessKeyKey,
			SecretKeyKey: override.Credentials.SecretRef.SecretKeyKey,
		}
	}
	return sessionInfo, nil
}

// getBucketAuthByPrefix returns first matching bucketname and prefix in authConfigs
func (p S3ProviderConfig) getBucketAuthByPrefix(bucketName, prefix string) *S3Override {
	for _, override := range p.Overrides {
		if override.BucketName == bucketName && strings.HasPrefix(prefix, override.KeyPrefix) {
			return &override
		}
	}
	return nil
}
