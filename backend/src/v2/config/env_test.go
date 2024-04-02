// Copyright 2024 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
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
	"github.com/stretchr/testify/assert"
	"os"
	"sigs.k8s.io/yaml"
	"testing"
)

type TestcaseData struct {
	Testcases []ProviderCase `json:"cases"`
}
type ProviderCase struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func Test_getDefaultMinioSessionInfo(t *testing.T) {
	actualDefaultSession, err := getDefaultMinioSessionInfo()
	assert.Nil(t, err)
	expectedDefaultSession := objectstore.SessionInfo{
		Provider: "minio",
		Params: map[string]string{
			"region":       "minio",
			"endpoint":     "minio-service.kubeflow:9000",
			"disableSsl":   "true",
			"fromEnv":      "false",
			"secretName":   "mlpipeline-minio-artifact",
			"accessKeyKey": "accesskey",
			"secretKeyKey": "secretkey",
		},
	}
	assert.Equal(t, expectedDefaultSession, actualDefaultSession)
}

func TestGetBucketSessionInfo(t *testing.T) {

	providersDataFile, err := os.ReadFile("testdata/provider_cases.yaml")
	if os.IsNotExist(err) {
		panic(err)
	}

	var providersData TestcaseData
	err = yaml.Unmarshal(providersDataFile, &providersData)
	if err != nil {
		panic(err)
	}

	tt := []struct {
		msg                 string
		config              Config
		expectedSessionInfo objectstore.SessionInfo
		pipelineroot        string
		// optional
		shouldError bool
		// optional
		errorMsg     string
		testDataCase string
	}{
		{
			msg:                 "invalid_unsupported_object_store_protocol",
			pipelineroot:        "unsupported://my-bucket/v2/artifacts",
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "unsupported Cloud bucket",
		},
		{
			msg:                 "invalid_unsupported_pipeline_root_format",
			pipelineroot:        "minio.unsupported.format",
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "unrecognized pipeline root format",
		},
		{
			msg:          "valid_no_providers",
			pipelineroot: "minio://my-bucket/v2/artifacts",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "minio",
				Params: map[string]string{
					"region":       "minio",
					"endpoint":     "minio-service.kubeflow:9000",
					"disableSsl":   "true",
					"fromEnv":      "false",
					"secretName":   "mlpipeline-minio-artifact",
					"accessKeyKey": "accesskey",
					"secretKeyKey": "secretkey",
				},
			},
		},
		{
			msg:                 "invalid_empty_minio_provider",
			pipelineroot:        "minio://my-bucket/v2/artifacts",
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "invalid provider config",
			testDataCase:        "case1",
		},
		{
			msg:                 "invalid_empty_minio_provider_no_override",
			pipelineroot:        "minio://my-bucket/v2/artifacts",
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "invalid provider config",
			testDataCase:        "case2",
		},
		{
			msg:                 "invalid_minio_provider_endpoint_only",
			pipelineroot:        "minio://my-bucket/v2/artifacts",
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "invalid provider config",
			testDataCase:        "case3",
		},
		{
			msg:                 "invalid_one_minio_provider_no_creds",
			pipelineroot:        "minio://my-bucket/v2/artifacts",
			expectedSessionInfo: objectstore.SessionInfo{},
			shouldError:         true,
			errorMsg:            "invalid provider config",
			testDataCase:        "case4",
		},
		{
			msg:          "valid_minio_provider_with_default_only",
			pipelineroot: "minio://my-bucket/v2/artifacts",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "minio",
				Params: map[string]string{
					"region":       "minio",
					"endpoint":     "minio-endpoint-5.com",
					"disableSSL":   "true",
					"fromEnv":      "false",
					"secretName":   "test-secret-5",
					"accessKeyKey": "test-accessKeyKey-5",
					"secretKeyKey": "test-secretKeyKey-5",
				},
			},
			testDataCase: "case5",
		},
		{
			msg:          "valid_pick_minio_provider",
			pipelineroot: "minio://minio-bucket-a/some/minio/path/a",
			expectedSessionInfo: objectstore.SessionInfo{
				Provider: "minio",
				Params: map[string]string{
					"region":       "minio-a",
					"endpoint":     "minio-endpoint-6.com",
					"disableSSL":   "true",
					"fromEnv":      "false",
					"secretName":   "minio-test-secret-6-a",
					"accessKeyKey": "minio-test-accessKeyKey-6-a",
					"secretKeyKey": "minio-test-secretKeyKey-6-a",
				},
			},
			testDataCase: "case6",
		},
		//		{
		//			msg:          "invalid_non_minio_should_require_secret",
		//			pipelineroot: "s3://my-bucket/v2/artifacts",
		//			config: Config{
		//				data: map[string]string{
		//					"providers": `
		//s3:
		// endpoint: s3.amazonaws.com
		// region: us-east-1
		// authConfigs: []
		//`,
		//				},
		//			},
		//			expectedSessionInfo: objectstore.SessionInfo{},
		//			shouldError:         true,
		//			errorMsg:            "Invalid provider config",
		//		},
		//		{
		//			msg:          "invalid_matching_prefix_should_require_secretref",
		//			pipelineroot: "minio://my-bucket/v2/artifacts/pick/this",
		//			config: Config{
		//				data: map[string]string{
		//					"providers": `
		//minio:
		// endpoint: minio.endpoint.com
		// region: minio
		// authConfigs:
		//   - bucketName: my-bucket
		//     keyPrefix: v2/artifacts/skip/this
		//     secretRef:
		//       secretName: minio-skip-this-secret
		//       accessKeyKey: minio_skip_this_access_key
		//       secretKeyKey: minio_skip_this_secret_key
		//   - bucketName: my-bucket
		//     keyPrefix: v2/artifacts/pick/this
		//`,
		//				},
		//			},
		//			expectedSessionInfo: objectstore.SessionInfo{},
		//			shouldError:         true,
		//			errorMsg:            "Invalid provider config",
		//		},
		//		{
		//			msg:          "valid_s3_with_secret",
		//			pipelineroot: "s3://my-bucket/v2/artifacts",
		//			config: Config{
		//				data: map[string]string{
		//					"providers": `
		//s3:
		// endpoint: s3.amazonaws.com
		// region: us-east-1
		// defaultProviderSecretRef:
		//   secretName: "s3-provider-secret"
		//   accessKeyKey: "different_access_key"
		//   secretKeyKey: "different_secret_key"
		//`,
		//				},
		//			},
		//			expectedSessionInfo: objectstore.SessionInfo{
		//				Region:       "us-east-1",
		//				Endpoint:     "s3.amazonaws.com",
		//				DisableSSL:   false,
		//				SecretName:   "s3-provider-secret",
		//				SecretKeyKey: "different_secret_key",
		//				AccessKeyKey: "different_access_key",
		//			},
		//		},
		//		{
		//			msg:          "valid_minio_first_matching_auth_config",
		//			pipelineroot: "minio://my-bucket/v2/artifacts/pick/this",
		//			config: Config{
		//				data: map[string]string{
		//					"providers": `
		//minio:
		// endpoint: minio.endpoint.com
		// region: minio
		// defaultProviderSecretRef:
		//     secretName: minio-default-provider-secret
		//     accessKeyKey: minio_default_different_access_key
		//     secretKeyKey: minio_default_different_secret_key
		// authConfigs:
		//   - bucketName: my-bucket
		//     keyPrefix: v2/artifacts/skip/this
		//     secretRef:
		//       secretName: minio-skip-this-secret
		//       accessKeyKey: minio_skip_this_access_key
		//       secretKeyKey: minio_skip_this_secret_key
		//   - bucketName: my-bucket
		//     keyPrefix: v2/artifacts/pick/this
		//     secretRef:
		//       secretName: minio-pick-this-secret
		//       accessKeyKey: minio_pick_this_access_key
		//       secretKeyKey: minio_pick_this_secret_key
		//   - bucketName: my-bucket
		//     keyPrefix: v2/artifacts
		//     secretRef:
		//       secretName: minio-not-reached-secret
		//       accessKeyKey: minio_not_reached_access_key
		//       secretKeyKey: minio_not_reached_secret_key
		//`,
		//				},
		//			},
		//			expectedSessionInfo: objectstore.SessionInfo{
		//				Region:       "minio",
		//				Endpoint:     "minio.endpoint.com",
		//				DisableSSL:   false,
		//				SecretName:   "minio-pick-this-secret",
		//				SecretKeyKey: "minio_pick_this_secret_key",
		//				AccessKeyKey: "minio_pick_this_access_key",
		//			},
		//		},
		//		{
		//			msg:          "valid_gs_first_matching_auth_config",
		//			pipelineroot: "gs://my-bucket/v2/artifacts/pick/this",
		//			config: Config{
		//				data: map[string]string{
		//					"providers": `
		//minio:
		// endpoint: minio.endpoint.com
		// region: minio
		// defaultProviderSecretRef:
		//     secretName: minio-default-provider-secret
		//     accessKeyKey: minio_default_different_access_key
		//     secretKeyKey: minio_default_different_secret_key
		// authConfigs:
		//   - bucketName: my-bucket
		//     keyPrefix: v2/artifacts/skip/this
		//     secretRef:
		//       secretName: minio-skip-this-secret
		//       accessKeyKey: minio_skip_this_access_key
		//       secretKeyKey: minio_skip_this_secret_key
		//gcs:
		// endpoint: storage.googleapis.com
		// region: gcs
		// defaultProviderSecretRef:
		//   secretName: gcs-default-provider-secret
		//   accessKeyKey: gcs_default_different_access_key
		//   secretKeyKey: gcs_default_different_secret_key
		// authConfigs:
		//   - bucketName: my-bucket
		//     keyPrefix: v2/artifacts/pick/this
		//     secretRef:
		//       secretName: gcs-pick-this-secret
		//       accessKeyKey: gcs_pick_this_access_key
		//       secretKeyKey: gcs_pick_this_secret_key
		//`,
		//				},
		//			},
		//			expectedSessionInfo: objectstore.SessionInfo{
		//				Region:       "gcs",
		//				Endpoint:     "storage.googleapis.com",
		//				DisableSSL:   false,
		//				SecretName:   "gcs-pick-this-secret",
		//				SecretKeyKey: "gcs_pick_this_secret_key",
		//				AccessKeyKey: "gcs_pick_this_access_key",
		//			},
		//		},
		//		{
		//			msg:          "valid_pick_default_when_no_matching_prefix",
		//			pipelineroot: "gs://my-bucket/v2/artifacts/pick/default",
		//			config: Config{
		//				data: map[string]string{
		//					"providers": `
		//gcs:
		// endpoint: storage.googleapis.com
		// region: gcs
		// defaultProviderSecretRef:
		//   secretName: gcs-default-provider-secret
		//   accessKeyKey: gcs_default_different_access_key
		//   secretKeyKey: gcs_default_different_secret_key
		// authConfigs:
		//   - bucketName: my-bucket
		//     keyPrefix: v2/artifacts/skip/this
		//     secretRef:
		//       secretName: gcs-skip-this-secret
		//       accessKeyKey: gcs_skip_this_access_key
		//       secretKeyKey: gcs_skip_this_secret_key
		//`,
		//				},
		//			},
		//			expectedSessionInfo: objectstore.SessionInfo{
		//				Region:       "gcs",
		//				Endpoint:     "storage.googleapis.com",
		//				DisableSSL:   false,
		//				SecretName:   "gcs-default-provider-secret",
		//				SecretKeyKey: "gcs_default_different_secret_key",
		//				AccessKeyKey: "gcs_default_different_access_key",
		//			},
		//		},
	}

	for _, test := range tt {
		t.Run(test.msg, func(t *testing.T) {
			config := Config{data: map[string]string{}}
			if test.testDataCase != "" {
				config.data["providers"] = fetchProviderFromdata(providersData, test.testDataCase)
				if config.data["providers"] == "" {
					panic(fmt.Errorf("provider not found in testdata"))
				}
			}

			actualSession, err := config.GetBucketSessionInfo(test.pipelineroot)
			if test.shouldError {
				assert.Error(t, err)
				if err != nil && test.errorMsg != "" {
					assert.Contains(t, err.Error(), test.errorMsg)
				}
			} else {
				assert.Nil(t, err)
			}

			assert.Equal(t, test.expectedSessionInfo, actualSession)
		})
	}
}

func fetchProviderFromdata(cases TestcaseData, name string) string {
	for _, c := range cases.Testcases {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}
