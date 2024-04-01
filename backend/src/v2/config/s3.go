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

import "github.com/kubeflow/pipelines/backend/src/v2/objectstore"

type S3ProviderConfig struct {
	Endpoint    string         `json:"endpoint"`
	Credentials *S3Credentials `json:"credentials"`
	// optional for any non aws s3 provider
	Region string `json:"region"`
	// optional
	DisableSSL bool `json:"disableSSL"`
	// optional, ordered, the auth config for the first matching prefix is used
	Overrides []S3Overrides `json:"Overrides"`
}
type S3Credentials struct {
	// optional
	FromeEnv  bool         `json:"fromEnv"`
	SecretRef *S3SecretRef `json:"secretRef"`
}
type S3Overrides struct {
	Endpoint    string         `json:"endpoint"`
	Region      string         `json:"region"`
	BucketName  string         `json:"bucketName"`
	KeyPrefix   string         `json:"keyPrefix"`
	Credentials *S3Credentials `json:"credentials"`
}
type S3SecretRef struct {
	SecretName   string `json:"secretName"`
	AccessKeyKey string `json:"accessKeyKey"`
	SecretKeyKey string `json:"secretKeyKey"`
}

func (s S3ProviderConfig) ProvideSessionInfo(bucketName, bucketPrefix string) (objectstore.SessionInfo, error) {
	//TODO implement me
	panic("implement me")
}
